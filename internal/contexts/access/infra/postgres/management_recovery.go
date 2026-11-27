package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"errors"
	"strings"

	"chnhcm.local/chnhcm/internal/contexts/access/domain"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RecoverManagement is only wired to the deployment command using the instance owner's database identity.
// expectedAdmin prevents a stale recovery or handover from replacing a later appointment.
func RecoverManagement(ctx context.Context, p *pgxpool.Pool, instance string, in access.RecoveryCommand) (v access.ManagementResult, err error) {
	login, ok := domain.NormalizeLogin(in.Login)
	if !ok || !foundation.ValidID(instance) || instance != in.Instance || !foundation.ValidID(in.Request) || strings.TrimSpace(in.Operator) == "" || len(in.Operator) > 180 || strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 1000 || in.MaxProxyDays != nil && (*in.MaxProxyDays < 0 || *in.MaxProxyDays > 3650) {
		return v, invalidManagement("请核对实例、目标账号、运维身份、事由及代理期限上限")
	}
	tx, e := p.Begin(ctx)
	if e != nil {
		return v, e
	}
	defer tx.Rollback(ctx)
	if e = lockManager(ctx, tx, instance, true); e != nil {
		return v, e
	}
	var current string
	var days int
	e = tx.QueryRow(ctx, "SELECT principal_id::text,max_proxy_days FROM access.management_root WHERE instance_id=$1", instance).Scan(&current, &days)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return v, e
	}
	if in.MaxProxyDays != nil {
		days = *in.MaxProxyDays
	}
	if current != "" {
		prior, old, e := readReceipt(ctx, tx, instance, "", in.Request)
		if e == nil {
			seal, e := commandSeal(ctx, tx, instance, in)
			if e != nil {
				return v, e
			}
			if !hmac.Equal(seal, old) {
				return v, changedManagement()
			}
			return prior, nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return v, e
		}
	}
	switch in.Operation {
	case "bootstrap":
		if current != "" || in.ExpectedAdmin != "" {
			return v, changedManagement()
		}
	case "recover", "handover":
		if current == "" || in.ExpectedAdmin != current {
			return v, changedManagement()
		}
	default:
		return v, invalidManagement("请选择首次引导、恢复或交接")
	}
	var id string
	created := false
	e = tx.QueryRow(ctx, "SELECT principal_id::text FROM access.principal WHERE instance_id=$1 AND normalized_login=$2", instance, strings.ToLower(login)).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		if in.Operation == "recover" {
			return v, changedManagement()
		}
		id, e = createManagedAccount(ctx, tx, instance, instance, login, "", in.Password)
		created = e == nil
	}
	if e != nil {
		return v, e
	}
	if in.Operation == "recover" && id != current || in.Operation == "handover" && id == current {
		return v, changedManagement()
	}
	if e = LockManagement(ctx, tx, instance, current, id, access.VersionBinding{}, "", ""); e != nil {
		return v, e
	}
	// A target with an existing credential may keep it on bootstrap or handover. Recovery can explicitly rotate it.
	if in.Password != "" && !created {
		if e = resetCredential(ctx, tx, instance, id, in.Password); e != nil {
			return v, e
		}
	}
	if _, e = tx.Exec(ctx, "UPDATE access.principal SET state='active' WHERE instance_id=$1 AND principal_id=$2", instance, id); e != nil {
		return v, e
	}
	if current == "" {
		key := make([]byte, 32)
		if _, e = rand.Read(key); e != nil {
			return v, e
		}
		_, e = tx.Exec(ctx, "INSERT INTO access.management_root(instance_id,principal_id,max_proxy_days,receipt_key) VALUES($1,$2,$3,$4)", instance, id, days, key)
	} else {
		_, e = tx.Exec(ctx, "UPDATE access.management_root SET principal_id=$2,max_proxy_days=$3 WHERE instance_id=$1", instance, id, days)
	}
	if e != nil {
		return v, e
	}
	if current != "" && current != id {
		if _, e = tx.Exec(ctx, "UPDATE access.management_mandate SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND principal_id=$2 AND revoked_at IS NULL", instance, current); e != nil {
			return v, e
		}
		if e = invalidateSessions(ctx, tx, instance, current); e != nil {
			return v, e
		}
		if e = bumpSubject(ctx, tx, instance, current); e != nil {
			return v, e
		}
		if e = recordManagement(ctx, tx, instance, "", current, in.Request, "admin.handover.end", managementChangeReason); e != nil {
			return v, e
		}
	}
	if e = invalidateSessions(ctx, tx, instance, id); e != nil {
		return v, e
	}
	if e = bumpSubject(ctx, tx, instance, id); e != nil {
		return v, e
	}
	v = access.ManagementResult{Operation: "admin." + in.Operation, Target: id, Subject: id, Request: in.Request}
	reason := managementChangeReason
	if in.Password != "" && !created {
		reason = rotatedCredentialReason
	}
	if e = recordManagement(ctx, tx, instance, "", id, in.Request, v.Operation, reason); e != nil {
		return v, e
	}
	seal, e := commandSeal(ctx, tx, instance, in)
	if e != nil {
		return v, e
	}
	if e = saveReceipt(ctx, tx, instance, "", in.Operator, in.Reason, seal, ownershipChange{}, &v); e != nil {
		return v, e
	}
	return v, tx.Commit(ctx)
}

// The same subject lock is held by the old deployment maintenance commands and by an appointment.
func protectRecoveryAccount(ctx context.Context, tx pgx.Tx, instance, id string) error {
	var held bool
	err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM access.management_root WHERE instance_id=$1 AND principal_id=$2)", instance, id).Scan(&held)
	if err == nil && held {
		return errors.New("受保护管理员须通过 access-admin 恢复或交接")
	}
	return err
}

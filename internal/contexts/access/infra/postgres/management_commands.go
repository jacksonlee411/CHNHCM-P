package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"chnhcm.local/chnhcm/internal/contexts/access/domain"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func commandSeal(ctx context.Context, tx pgx.Tx, instance string, body any) ([]byte, error) {
	var key []byte
	if e := tx.QueryRow(ctx, "SELECT receipt_key FROM access.management_root WHERE instance_id=$1", instance).Scan(&key); e != nil {
		return nil, e
	}
	raw, e := json.Marshal(body)
	if e != nil {
		return nil, e
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(raw)
	return mac.Sum(nil), nil
}
func readReceipt(ctx context.Context, tx pgx.Tx, instance, actor, request string) (v access.ManagementResult, seal []byte, err error) {
	err = tx.QueryRow(ctx, `SELECT request_id::text,operation,target_id::text,coalesce(subject_id::text,''),recorded_at,command_seal FROM access.management_receipt WHERE instance_id=$1 AND actor_id IS NOT DISTINCT FROM $2::uuid AND request_id=$3`, instance, nullable(actor), request).Scan(&v.Request, &v.Operation, &v.Target, &v.Subject, &v.At, &seal)
	return
}

type ownershipChange struct{ previous, assigned string }

func saveReceipt(ctx context.Context, tx pgx.Tx, instance, actor, operator, reason string, seal []byte, owners ownershipChange, v *access.ManagementResult) error {
	return tx.QueryRow(ctx, `INSERT INTO access.management_receipt(instance_id,actor_id,operator_name,request_id,operation,target_id,subject_id,reason,command_seal,previous_owner,assigned_owner)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING recorded_at`, instance, nullable(actor), nullable(operator), v.Request, v.Operation, v.Target, nullable(v.Subject), reason, seal, nullable(owners.previous), nullable(owners.assigned)).Scan(&v.At)
}
func (m *manager) canReadResult(v access.ManagementResult) error {
	m.checking = true
	if v.Operation == "submission.closed" && (m.areaOK("grants") || m.areaOK("roles") || m.areaOK("accounts")) {
		return nil
	}
	if !m.areaOK(areaFor(v.Operation)) {
		return managementDenied()
	}
	if m.root {
		return nil
	}
	if v.Operation == "account.reset" && !m.permits("credential", "", nil) {
		return managementDenied()
	}
	if v.Operation == "account.create" || v.Operation == "account.enable" || v.Operation == "account.move" || v.Operation == "account.disable" {
		if !m.permits("account", "", nil) {
			return managementDenied()
		}
	}
	q := access.ManagementQuery{Area: areaFor(v.Operation), Subject: v.Subject}
	if q.Area == "roles" {
		q.Object = v.Target
	}
	view, e := m.view(q)
	if e != nil {
		return e
	}
	if areaFor(v.Operation) == "accounts" && len(view.Accounts) == 1 {
		right := "account"
		if v.Operation == "account.reset" {
			right = "credential"
		}
		if !m.ownerScope(right, view.Accounts[0].Owner) {
			return managementDenied()
		}
	}
	if !slices.ContainsFunc(view.Events, func(event access.ManagementEvent) bool { return event.Request == v.Request }) {
		return managementDenied()
	}
	return nil
}
func (s *Store) ManagementResult(ctx context.Context, c access.Credentials, request string) (v access.ManagementResult, err error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return v, e
	}
	defer tx.Rollback(ctx)
	m, g, e := s.manager(ctx, tx, c, nil)
	if e != nil {
		return v, e
	}
	if !foundation.ValidID(request) {
		return v, managementDenied()
	}
	v, _, e = readReceipt(ctx, tx, m.instance, m.actor, request)
	if errors.Is(e, pgx.ErrNoRows) {
		return v, foundation.Fail(404, "ACCESS_RECEIPT_PENDING", "尚未查到本次提交")
	}
	if e != nil {
		return v, e
	}
	if e = m.canReadResult(v); e != nil {
		return v, e
	}
	return v, m.finish(g)
}
func (s *Store) Manage(ctx context.Context, c access.Credentials, in access.ManagementCommand) (v access.ManagementResult, err error) {
	if in.Instance != s.Instance {
		return v, managementDenied()
	}
	if !foundation.ValidID(in.Request) || len(in.Reason) > 1000 {
		return v, invalidManagement("提交身份或事由无效")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return v, e
	}
	defer tx.Rollback(ctx)
	c.Write = true
	m, g, e := s.manager(ctx, tx, c, &in)
	if e != nil {
		return v, e
	}
	if !m.areaOK(areaFor(in.Operation)) {
		return v, managementDenied()
	}
	seal, e := commandSeal(ctx, tx, s.Instance, in)
	if e != nil {
		return v, e
	}
	prior, old, e := readReceipt(ctx, tx, s.Instance, m.actor, in.Request)
	if e == nil {
		if e = m.canReadResult(prior); e != nil {
			return v, e
		}
		if !hmac.Equal(seal, old) {
			return v, changedManagement()
		}
		return prior, m.finish(g)
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return v, e
	}
	if e = m.prepare(); e != nil {
		return v, e
	}
	if e = m.directory(); e != nil {
		return v, e
	}
	v = access.ManagementResult{Operation: in.Operation, Request: in.Request, Subject: in.Subject}
	var owners ownershipChange
	m.checking = true
	switch areaFor(in.Operation) {
	case "accounts":
		v.Target, e = m.changeAccount(in, &owners)
		v.Subject = v.Target
	case "roles":
		v.Target, e = m.changeVersion(in)
	case "grants":
		v.Target, e = m.changeGrant(in)
	}
	if e != nil {
		return v, managementError(e)
	}
	if e = saveReceipt(ctx, tx, m.instance, m.actor, "", in.Reason, seal, owners, &v); e != nil {
		return v, managementError(e)
	}
	return v, m.finish(g)
}
func managementError(e error) error {
	var db *pgconn.PgError
	if errors.As(e, &db) {
		switch db.Code {
		case "23505":
			return changedManagement()
		case "23514", "22P02", "22001":
			return invalidManagement("请核对所选版本、范围及字段")
		}
	}
	return e
}

const (
	managementChangeReason  = "MANAGEMENT_CHANGE"
	rotatedCredentialReason = "CREDENTIAL_ROTATED"
)

func recordManagement(ctx context.Context, tx pgx.Tx, instance, actor, subject, request, op, reason string) error {
	_, e := tx.Exec(ctx, `INSERT INTO access.security_event(instance_id,event_id,event_kind,occurred_at,action,decision,reason_code,actor_subject_id,target_subject_id,request_id)
 VALUES($1,$2,'security_change',clock_timestamp(),$3,'allow',$7,$4,$5,$6)`, instance, foundation.NewID(), op, nullable(actor), nullable(subject), request, reason)
	return e
}
func passwordOK(p string) bool { return len(p) >= 12 && len(p) <= 1024 }
func resetCredential(ctx context.Context, tx pgx.Tx, instance, id, password string) error {
	if !passwordOK(password) {
		return invalidManagement("凭据长度须为 12 至 1024 字节")
	}
	_, salt := token()
	salt = salt[:16]
	hash, e := pbkdf2.Key(sha256.New, password, salt, 600000, 32)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `UPDATE access.credential SET salt=$3,password_hash=$4,iterations=600000,updated_at=clock_timestamp() WHERE instance_id=$1 AND principal_id=$2`, instance, id, salt, hash)
	return e
}
func invalidateSessions(ctx context.Context, tx pgx.Tx, instance, id string) error {
	_, e := tx.Exec(ctx, `UPDATE access.session SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND principal_id=$2 AND revoked_at IS NULL`, instance, id)
	return e
}
func bumpSubject(ctx context.Context, tx pgx.Tx, instance, id string) error {
	_, e := tx.Exec(ctx, `UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2`, instance, id)
	return e
}
func createManagedAccount(ctx context.Context, tx pgx.Tx, instance, owner, login, display, password string) (string, error) {
	login, ok := domain.NormalizeLogin(login)
	if !ok || len(display) > 200 || !passwordOK(password) {
		return "", invalidManagement("请核对登录名、显示名及初始凭据（12 至 1024 字节）")
	}
	id := foundation.NewID()
	// A managed account has a distinct creation identity and can never match a development initialization declaration.
	_, e := tx.Exec(ctx, `INSERT INTO access.principal(instance_id,principal_id,login,normalized_login,display_name,state,initialization_digest,management_owner)
 VALUES($1,$2,$3,$4,$5,'active',$6,$7)`, instance, id, login, strings.ToLower(login), nullable(strings.TrimSpace(display)), digest(id), owner)
	if e != nil {
		return "", e
	}
	_, salt := token()
	salt = salt[:16]
	hash, e := pbkdf2.Key(sha256.New, password, salt, 600000, 32)
	if e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO access.credential(instance_id,principal_id,algorithm,parameter_version,iterations,salt,password_hash) VALUES($1,$2,'pbkdf2-sha256',1,600000,$3,$4)`, instance, id, salt, hash); e != nil {
		return "", e
	}
	_, e = tx.Exec(ctx, "INSERT INTO access.policy_guard VALUES($1,$2,1)", instance, id)
	return id, e
}
func (m *manager) changeAccount(in access.ManagementCommand, owners *ownershipChange) (id string, err error) {
	if in.Operation == "account.create" {
		if !m.ownerAllowed("account", in.Owner) {
			return "", managementDenied()
		}
		owners.assigned = in.Owner
		id, err = createManagedAccount(m.ctx, m.tx, m.instance, in.Owner, in.Login, in.Display, in.Password)
		if err != nil {
			return "", err
		}
	} else {
		if !foundation.ValidID(in.Subject) {
			return "", managementDenied()
		}
		a, e := m.account(in.Subject)
		if e != nil {
			return "", e
		}
		id = a.ID
		right := "account"
		if in.Operation == "account.reset" {
			right = "credential"
		}
		ok, e := m.controls(a, right)
		if e != nil {
			return "", e
		}
		if !ok {
			return "", managementDenied()
		}
		if a.Revision != in.Revision {
			return "", changedManagement()
		}
		switch in.Operation {
		case "account.reset":
			err = resetCredential(m.ctx, m.tx, m.instance, id, in.Password)
		case "account.enable", "account.disable":
			state := "active"
			if in.Operation == "account.disable" {
				state = "disabled"
			}
			if state == a.State {
				return "", changedManagement()
			}
			_, err = m.tx.Exec(m.ctx, "UPDATE access.principal SET state=$3 WHERE instance_id=$1 AND principal_id=$2", m.instance, id, state)
		case "account.move":
			if !m.ownerAllowed("account", in.Owner) {
				return "", managementDenied()
			}
			if in.Owner == a.Owner {
				return "", changedManagement()
			}
			owners.previous, owners.assigned = a.Owner, in.Owner
			newOwner := a
			newOwner.Owner = in.Owner
			ok, e := m.controls(newOwner, "account")
			if e != nil {
				return "", e
			}
			if !ok {
				return "", managementDenied()
			}
			_, err = m.tx.Exec(m.ctx, "UPDATE access.principal SET management_owner=$3 WHERE instance_id=$1 AND principal_id=$2", m.instance, id, in.Owner)
		default:
			return "", invalidManagement("未知账号操作")
		}
		if err != nil {
			return "", err
		}
		if in.Operation == "account.reset" || in.Operation == "account.disable" {
			if err = invalidateSessions(m.ctx, m.tx, m.instance, id); err != nil {
				return "", err
			}
		}
		if err = bumpSubject(m.ctx, m.tx, m.instance, id); err != nil {
			return "", err
		}
	}
	reason := managementChangeReason
	if in.Operation == "account.reset" {
		reason = rotatedCredentialReason
	}
	return id, recordManagement(m.ctx, m.tx, m.instance, m.actor, id, in.Request, in.Operation, reason)
}

// ResolveManagement fences a submission that never reached the server under the same lock as its write.
func (s *Store) ResolveManagement(ctx context.Context, c access.Credentials, id string) (v access.ManagementResult, err error) {
	if !foundation.ValidID(id) {
		return v, invalidManagement("提交身份无效")
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return v, e
	}
	defer tx.Rollback(ctx)
	c.Write = true
	m, g, e := s.manager(ctx, tx, c, &access.ManagementCommand{})
	if e != nil {
		return v, e
	}
	v, _, e = readReceipt(ctx, tx, s.Instance, m.actor, id)
	if errors.Is(e, pgx.ErrNoRows) {
		v = access.ManagementResult{Request: id, Operation: "submission.closed", Target: id}
		if e = m.canReadResult(v); e != nil {
			return v, e
		}
		e = saveReceipt(ctx, tx, s.Instance, m.actor, "", "", digest("closed"), ownershipChange{}, &v)
		var conflict *pgconn.PgError
		if errors.As(e, &conflict) && conflict.Code == "23505" {
			return v, changedManagement()
		}
	} else if e == nil {
		e = m.canReadResult(v)
	}
	if e != nil {
		return v, e
	}
	return v, m.finish(g)
}

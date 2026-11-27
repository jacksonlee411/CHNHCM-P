package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func purposeFor(action string) string {
	purpose, _, _ := strings.Cut(action, ".")
	return purpose
}

const publishVersionSQL = `INSERT INTO access.policy_version(instance_id,kind,version_id,definition_id,version_no,actions,scope_kind,scope_id,read_fields,write_fields)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`

// PublishVersion and DisableVersion require a management transaction. The
// runtime DB role has only read/lock access until T04 admits its command surface.
func PublishVersion(ctx context.Context, tx pgx.Tx, instance, actor, request string, v access.PolicyVersion) error {
	if !foundation.ValidID(request) || !foundation.ValidID(actor) {
		return errors.New("missing publication identity")
	}
	if err := LockManagement(ctx, tx, instance, actor, "", access.VersionBinding{}, "", ""); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, publishVersionSQL, instance, v.Kind, v.ID, v.DefinitionID, v.Number, v.Actions, nullable(v.ScopeKind), nullable(v.ScopeID), v.ReadFields, v.WriteFields)
	if err != nil {
		return err
	}
	return versionEvent(ctx, tx, instance, actor, request, v.Kind, v.ID, "publish")
}

func DisableVersion(ctx context.Context, tx pgx.Tx, instance, actor, request, kind, id string) error {
	if !foundation.ValidID(request) || !foundation.ValidID(actor) {
		return errors.New("missing retirement identity")
	}
	if err := LockManagement(ctx, tx, instance, actor, "", access.VersionBinding{}, kind, id); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE access.policy_version SET disabled_at=clock_timestamp()
 WHERE instance_id=$1 AND kind=$2 AND version_id=$3 AND disabled_at IS NULL`, instance, kind, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("version missing or already disabled")
	}
	return versionEvent(ctx, tx, instance, actor, request, kind, id, "disable")
}
func versionEvent(ctx context.Context, tx pgx.Tx, instance, actor, request, kind, id, op string) error {
	_, err := tx.Exec(ctx, `INSERT INTO access.security_event(instance_id,event_id,event_kind,occurred_at,action,decision,reason_code,actor_subject_id,request_id)
 VALUES($1,$2,'security_change',clock_timestamp(),$3,'allow',$4,$5,$6)`, instance, foundation.NewID(), "version."+op+"."+kind, id, actor, request)
	return err
}

const issueAssignmentSQL = `INSERT INTO access.grant_assignment(instance_id,grant_id,principal_id,source,purpose,valid_from,valid_until,revision)
 VALUES($1,$2,$3,$4,$5,$6,$7,1)`
const appendRevisionSQL = `INSERT INTO access.grant_revision(instance_id,grant_id,revision,role_id,scope_id,fields_id,operation,actor_id,request_id,recorded_at,command)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,clock_timestamp(),$10)`

// ChangeAssignment serializes one subject's mutations. Every business request
// locks the same row before resolving its version dependencies.
func ChangeAssignment(ctx context.Context, tx pgx.Tx, instance string, c access.AssignmentChange) (int64, error) {
	if !foundation.ValidID(c.Actor) {
		return 0, errors.New("missing administrator identity")
	}
	return changeAssignment(ctx, tx, instance, c)
}

func changeAssignment(ctx context.Context, tx pgx.Tx, instance string, c access.AssignmentChange) (int64, error) {
	if !foundation.ValidID(c.RequestID) || !foundation.ValidID(c.ID) {
		return 0, errors.New("missing assignment identity")
	}
	if err := LockManagement(ctx, tx, instance, c.Actor, c.Subject, c.Binding, "", ""); err != nil {
		return 0, err
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return 0, err
	}
	var revision int64
	var same bool
	err = tx.QueryRow(ctx, `SELECT revision,command=$3::jsonb FROM access.grant_revision WHERE instance_id=$1 AND request_id=$2`, instance, c.RequestID, payload).Scan(&revision, &same)
	if err == nil {
		if !same {
			return 0, errors.New("submission already used with different content")
		}
		return revision, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	binding := c.Binding
	purpose := c.Purpose
	switch c.Operation {
	case "issue":
		if c.ExpectedRevision != 0 || (c.Source != "duty" && c.Source != "proxy") || c.ValidFrom == nil {
			return 0, errors.New("invalid initial assignment")
		}
	case "adopt", "revoke":
		var previous access.VersionBinding
		var revoked, expired bool
		err = tx.QueryRow(ctx, `SELECT a.revision,r.role_id,r.scope_id,r.fields_id,r.operation='revoke',coalesce(a.valid_until<=clock_timestamp(),false),a.purpose
 FROM access.grant_assignment a JOIN access.grant_revision r USING(instance_id,grant_id,revision)
 WHERE a.instance_id=$1 AND a.principal_id=$2 AND a.grant_id=$3`, instance, c.Subject, c.ID).Scan(&revision, &previous.Role, &previous.Scope, &previous.Fields, &revoked, &expired, &purpose)
		if err != nil {
			return 0, err
		}
		if revoked || revision != c.ExpectedRevision || c.Operation == "adopt" && expired {
			return 0, errors.New("assignment revision no longer available")
		}
		if c.Source != "" || c.Purpose != "" || c.ValidFrom != nil || c.ValidUntil != nil {
			return 0, errors.New("assignment terms are fixed")
		}
		if c.Operation == "revoke" {
			if binding != (access.VersionBinding{}) {
				return 0, errors.New("revocation does not take a binding")
			}
			binding = previous
		} else {
			if binding == previous {
				return 0, errors.New("assignment already uses these versions")
			}
			var related bool
			err = tx.QueryRow(ctx, `SELECT count(*)=3 FROM (VALUES ('role',$2::uuid,$3::uuid),('scope',$4,$5),('fields',$6,$7)) ref(kind,old_id,new_id)
 JOIN access.policy_version a ON (a.instance_id,a.kind,a.version_id)=($1,ref.kind,ref.old_id)
 JOIN access.policy_version b ON (b.instance_id,b.kind,b.version_id)=(a.instance_id,a.kind,ref.new_id)
 WHERE a.definition_id=b.definition_id`, instance, previous.Role, binding.Role, previous.Scope, binding.Scope, previous.Fields, binding.Fields).Scan(&related)
			if err != nil {
				return 0, err
			}
			if !related {
				return 0, errors.New("adoption requires versions of the bound definitions")
			}
		}
	default:
		return 0, errors.New("unknown assignment change")
	}
	if c.Operation != "revoke" {
		if err = lockBinding(ctx, tx, instance, binding, purpose); err != nil {
			return 0, err
		}
	}
	revision++
	if c.Operation == "issue" {
		_, err = tx.Exec(ctx, issueAssignmentSQL, instance, c.ID, c.Subject, c.Source, purpose, c.ValidFrom, c.ValidUntil)
	} else {
		_, err = tx.Exec(ctx, `UPDATE access.grant_assignment SET revision=$3 WHERE instance_id=$1 AND grant_id=$2`, instance, c.ID, revision)
	}
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, appendRevisionSQL, instance, c.ID, revision, binding.Role, binding.Scope, binding.Fields, c.Operation, nullable(c.Actor), c.RequestID, payload)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO access.security_event(instance_id,event_id,event_kind,occurred_at,action,decision,reason_code,actor_subject_id,target_subject_id,request_id)
 VALUES($1,$2,'security_change',clock_timestamp(),$3,'allow',$4,$5,$6,$7)`, instance, foundation.NewID(), "assignment."+c.Operation, c.ID, nullable(c.Actor), c.Subject, c.RequestID)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2`, instance, c.Subject)
	return revision, err
}

func lockBinding(ctx context.Context, tx pgx.Tx, instance string, b access.VersionBinding, purpose string) error {
	rows, err := tx.Query(ctx, `SELECT kind,actions,disabled_at IS NULL FROM access.policy_version WHERE instance_id=$1
 AND (kind,version_id) IN (('fields',$2::uuid),('role',$3::uuid),('scope',$4::uuid)) ORDER BY kind,version_id FOR SHARE`, instance, b.Fields, b.Role, b.Scope)
	if err != nil {
		return err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var kind string
		var actions []string
		var enabled bool
		if err = rows.Scan(&kind, &actions, &enabled); err != nil {
			return err
		}
		if !enabled || kind == "role" && !slices.ContainsFunc(actions, func(action string) bool { return purposeFor(action) == purpose }) {
			return errors.New("inactive or inapplicable policy version")
		}
		n++
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	if n != 3 {
		return errors.New("incomplete policy binding")
	}
	return nil
}

// Development declarations share the same binding and revision format. The
// principal is new and uncommitted, so no competing holder can see these rows.
func queueDevelopmentGrant(batch *pgx.Batch, instance, subject, kind, scope, action string, read, write []string) {
	queueInitialGrant(batch, instance, foundation.NewID(), subject, kind, scope, action, read, write, "development")
}

func queueInitialGrant(batch *pgx.Batch, instance, id, subject, kind, scope, action string, read, write []string, source string) {
	batch.Queue(publishVersionSQL, instance, "role", id, id, 1, []string{action}, nil, nil, nil, nil)
	batch.Queue(publishVersionSQL, instance, "scope", id, id, 1, nil, kind, scope, nil, nil)
	batch.Queue(publishVersionSQL, instance, "fields", id, id, 1, nil, nil, nil, read, write)
	batch.Queue(issueAssignmentSQL, instance, id, subject, source, purposeFor(action), nil, nil)
	if source == "migration" {
		batch.Queue(`INSERT INTO access.grant_revision(instance_id,grant_id,revision,role_id,scope_id,fields_id,operation) VALUES($1,$2,1,$2,$2,$2,'issue')`, instance, id)
	} else {
		batch.Queue(appendRevisionSQL, instance, id, 1, id, id, id, "issue", nil, foundation.NewID(), nil)
	}
}

// RestoreGrantArchive converts a batch from a frozen pre-00029 benchmark.
// Its caller verifies archive ownership and checksums in an isolated database.
func RestoreGrantArchive(ctx context.Context, tx pgx.Tx, body []byte) error {
	var rows []struct {
		Instance  string   `json:"instance_id"`
		ID        string   `json:"grant_id"`
		Principal string   `json:"principal_id"`
		Kind      string   `json:"scope_kind"`
		Scope     string   `json:"scope_id"`
		Action    string   `json:"action"`
		Read      []string `json:"read_fields"`
		Write     []string `json:"write_fields"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&rows); err != nil {
		return err
	}
	var batch pgx.Batch
	for _, r := range rows {
		queueInitialGrant(&batch, r.Instance, r.ID, r.Principal, r.Kind, r.Scope, r.Action, r.Read, r.Write, "migration")
	}
	return tx.SendBatch(ctx, &batch).Close()
}

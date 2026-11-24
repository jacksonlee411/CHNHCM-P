//go:build integration

package history

import (
	"context"
	"encoding/binary"
	"fmt"

	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
)

// MutateIntegrationSource is a privileged corruption fixture. It retains valid
// physical hashes so tests exercise semantic checks rather than checksum errors.
// It is absent from production builds. The caller must own the disposable DB.
func MutateIntegrationSource(ctx context.Context, tx pgx.Tx, kind Kind, instance, object, source string, change func(compact.Values, []byte, []byte, []byte) (Encoder, error)) error {
	n, e := Locate(ctx, tx, kind, instance, object, source)
	if e != nil {
		return e
	}
	c, e := LoadChunk(ctx, tx, kind, instance, object, n)
	if e != nil {
		return e
	}
	i := c.Index(source)
	state, e := LoadShape(ctx, tx, kind, instance, c.Enterprise, c.States[i])
	if e != nil {
		return e
	}
	operation, e := LoadShape(ctx, tx, kind, instance, c.Enterprise, c.Operations[i])
	if e != nil {
		return e
	}
	proof, e := LoadShape(ctx, tx, kind, instance, c.Enterprise, c.Evidence[i])
	if e != nil {
		return e
	}
	encode, e := change(c.Values, state, operation, proof)
	if e != nil {
		return e
	}
	p := compact.NewPool(c.Values)
	state, operation, proof, e = encode(p)
	if e != nil {
		return e
	}
	c.Values = p.Values
	c.States[i], e = putShape(ctx, tx, kind, instance, c.Enterprise, state)
	if e != nil {
		return e
	}
	c.Operations[i], e = putShape(ctx, tx, kind, instance, c.Enterprise, operation)
	if e != nil {
		return e
	}
	c.Evidence[i], e = putShape(ctx, tx, kind, instance, c.Enterprise, proof)
	if e != nil {
		return e
	}
	c.Dictionary, e = putDictionary(ctx, tx, kind, instance, c.Enterprise, c.Values)
	if e != nil {
		return e
	}
	c.Envelope, e = c.digest()
	if e != nil {
		return e
	}
	owner, key := "business_unit", "bu_id"
	if kind == Department {
		owner, key = "department", "department_id"
	}
	for _, table := range []string{kind.prefix() + "_history", owner} {
		if _, e = tx.Exec(ctx, "ALTER TABLE organization."+table+" DISABLE TRIGGER USER"); e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, fmt.Sprintf(`UPDATE organization.%s_history SET ids=$4::text[]::uuid[],dictionary_id=$5,states=$6,operations=$7,evidence=$8,envelope=$9 WHERE instance_id=$1 AND object_id=$2 AND chunk_no=$3`, kind.prefix()), instance, object, n, c.Values.IDs, c.Dictionary, c.States, c.Operations, c.Evidence, c.Envelope)
	if e != nil {
		return e
	}
	var head string
	if e = tx.QueryRow(ctx, fmt.Sprintf(`SELECT head_source_id::text FROM organization.%s WHERE instance_id=$1 AND %s=$2`, owner, key), instance, object).Scan(&head); e != nil {
		return e
	}
	// The shared pool grew even when only an old source changed. Rebuild the head
	// projection against its unchanged logical state and new dictionary digest.
	j := c.Index(head)
	if j >= 0 {
		current, e := LoadShape(ctx, tx, kind, instance, c.Enterprise, c.States[j])
		if e != nil {
			return e
		}
		if kind == BusinessUnit {
			v, e := DecodeBUState(ctx, c.Values, current)
			if e != nil {
				return e
			}
			e = SaveProjection(ctx, tx, c, head, func(p *compact.Pool) ([]byte, error) { return EncodeBUState(ctx, p, v) })
			if e != nil {
				return e
			}
		} else {
			v, e := DecodeDepartmentState(ctx, c.Values, current)
			if e != nil {
				return e
			}
			e = SaveProjection(ctx, tx, c, head, func(p *compact.Pool) ([]byte, error) { return EncodeDepartmentState(ctx, p, v) })
			if e != nil {
				return e
			}
		}
	}
	for _, table := range []string{kind.prefix() + "_history", owner} {
		if _, e = tx.Exec(ctx, "ALTER TABLE organization."+table+" ENABLE TRIGGER USER"); e != nil {
			return e
		}
	}
	return nil
}

func MutateIntegrationTerminal(ctx context.Context, tx pgx.Tx, kind Kind, instance, subject, submission string, change func(compact.Values, []byte) ([]byte, compact.Values, error)) error {
	v, b, e := LoadTerminal(ctx, tx, kind, instance, subject, submission)
	if e != nil {
		return e
	}
	raw, values, e := change(v, b)
	if e != nil {
		return e
	}
	var ent string
	meta, e := TerminalIdentity(ctx, tx, kind, instance, subject, submission)
	if e != nil {
		return e
	}
	e = tx.QueryRow(ctx, fmt.Sprintf(`SELECT enterprise_id::text FROM organization.%s_terminal WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3`, kind.prefix()), instance, subject, submission).Scan(&ent)
	if e != nil {
		return e
	}
	root, e := putShape(ctx, tx, kind, instance, ent, raw)
	if e != nil {
		return e
	}
	dict, e := putDictionary(ctx, tx, kind, instance, ent, values)
	if e != nil {
		return e
	}
	hash, e := compact.Digest([]string{kind.prefix(), instance, ent, subject, submission, meta.Object, meta.Outcome, "terminal"}, values, [][]byte{binary.BigEndian.AppendUint32(nil, uint32(root))})
	if e != nil {
		return e
	}
	table := "organization." + kind.prefix() + "_terminal"
	if _, e = tx.Exec(ctx, "ALTER TABLE "+table+" DISABLE TRIGGER immutable_terminal"); e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `UPDATE `+table+` SET ids=$4::text[]::uuid[],dictionary_id=$5,operation=$6,envelope=$7 WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3`, instance, subject, submission, values.IDs, dict, root, hash)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "ALTER TABLE "+table+" ENABLE TRIGGER immutable_terminal")
	return e
}

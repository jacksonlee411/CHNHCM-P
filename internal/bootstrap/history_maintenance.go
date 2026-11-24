package bootstrap

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HistoryMaintenanceInput is exclusively for an ownership-checked offline
// tool. Product HTTP handlers never invoke maintenance or prune dictionaries.
type HistoryMaintenanceInput struct {
	Instance       string
	OwnedInstances []string
	DryRun         bool
}

type DictionaryGarbage struct {
	Kind             string `json:"kind"`
	Rows             int64  `json:"rows"`
	StoredTupleBytes int64  `json:"storedTupleBytes"`
}

type HistoryMaintenanceResult struct {
	DryRun         bool                `json:"dryRun"`
	Before         []DictionaryGarbage `json:"before"`
	After          []DictionaryGarbage `json:"after"`
	Removed        int64               `json:"unreferencedDictionariesRemoved"`
	ProtectedRoots string              `json:"protectedRootsSHA256"`
	Seconds        float64             `json:"seconds"`
}

func maintenanceQuiescent(ctx context.Context, tx pgx.Tx) error {
	// PostgreSQL caches activity/statistics fetched within a transaction. A
	// second query without clearing that cache can miss a newly opened client.
	if _, err := tx.Exec(ctx, `SELECT pg_stat_clear_snapshot()`); err != nil {
		return err
	}
	var other int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND backend_type='client backend' AND pid<>pg_backend_pid()`).Scan(&other)
	if err != nil {
		return err
	}
	if other != 0 {
		return errors.New("offline history maintenance requires all other client connections to close")
	}
	return nil
}

func dictionaryGarbage(ctx context.Context, tx pgx.Tx, instance string) ([]DictionaryGarbage, error) {
	out := make([]DictionaryGarbage, 0, 2)
	for _, prefix := range []string{"bu", "department"} {
		v := DictionaryGarbage{Kind: prefix}
		err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT count(*),coalesce(sum(pg_column_size(d)),0) FROM organization.%[1]s_dictionary d WHERE d.instance_id=$1
 AND NOT EXISTS(SELECT FROM organization.%[1]s_history h WHERE h.dictionary_id=d.id)
 AND NOT EXISTS(SELECT FROM organization.%[1]s_terminal t WHERE t.dictionary_id=d.id)
 AND NOT EXISTS(SELECT FROM organization.%[1]s_read_segment r WHERE r.dictionary_id=d.id)`, prefix), instance).Scan(&v.Rows, &v.StoredTupleBytes)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// Each history envelope binds its complete source directory and value pool.
// Include every immutable shape and every referenced dictionary as well as the
// native object, relation and terminal roots. Only unreferenced dictionaries
// may disappear. Full semantic readback is a separate verification gate.
func historyMaintenanceRoots(ctx context.Context, tx pgx.Tx, instance string) (string, error) {
	parts := []string{}
	for _, prefix := range []string{"bu", "department"} {
		object := "business_unit"
		if prefix == "department" {
			object = "department"
		}
		identity := "bu_id"
		if prefix == "department" {
			identity = "department_id"
		}
		parts = append(parts, fmt.Sprintf(`SELECT '%[1]s_history' k,object_id::text||':'||chunk_no::text id,encode(envelope,'hex') value FROM organization.%[1]s_history WHERE instance_id=$1
 UNION ALL SELECT '%[1]s_terminal',subject_id::text||':'||submission_id::text,encode(envelope,'hex') FROM organization.%[1]s_terminal WHERE instance_id=$1
 UNION ALL SELECT '%[1]s_shape',id::text,encode(digest,'hex') FROM organization.%[1]s_shape WHERE instance_id=$1
 UNION ALL SELECT '%[1]s_dictionary',d.id::text,encode(d.digest,'hex') FROM organization.%[1]s_dictionary d WHERE instance_id=$1 AND
 (EXISTS(SELECT FROM organization.%[1]s_history h WHERE h.dictionary_id=d.id) OR EXISTS(SELECT FROM organization.%[1]s_terminal t WHERE t.dictionary_id=d.id) OR EXISTS(SELECT FROM organization.%[1]s_read_segment r WHERE r.dictionary_id=d.id))
 UNION ALL SELECT '%[2]s',%[3]s::text,encode(sha256(convert_to(to_jsonb(o)::text,'UTF8')),'hex') FROM organization.%[2]s o WHERE instance_id=$1
 UNION ALL SELECT '%[1]s_read_segment',object_id::text||':'||slot::text,encode(envelope,'hex') FROM organization.%[1]s_read_segment WHERE instance_id=$1`, prefix, object, identity))
	}
	parts = append(parts, `SELECT 'department_relation',department_id::text,encode(sha256(convert_to(to_jsonb(r)::text,'UTF8')),'hex') FROM organization.department_hierarchy_relation r WHERE instance_id=$1`)
	var digest []byte
	err := tx.QueryRow(ctx, `SELECT sha256(convert_to(coalesce(string_agg(k||':'||id||':'||value,E'\n' ORDER BY k,id),''),'UTF8')) FROM (`+strings.Join(parts, " UNION ALL ")+`) roots`, instance).Scan(&digest)
	return hex.EncodeToString(digest), err
}

func MaintainHistory(ctx context.Context, pool *pgxpool.Pool, input HistoryMaintenanceInput) (HistoryMaintenanceResult, error) {
	started := time.Now()
	result := HistoryMaintenanceResult{DryRun: input.DryRun}
	if !foundation.ValidID(input.Instance) || !slices.Contains(input.OwnedInstances, input.Instance) {
		return result, errors.New("explicit owned maintenance instance required")
	}
	for _, id := range input.OwnedInstances {
		if !foundation.ValidID(id) {
			return result, errors.New("invalid maintenance ownership")
		}
	}
	if err := CheckCurrentStorage(ctx, pool); err != nil {
		return result, err
	}
	options := pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
	if input.DryRun {
		options = pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	}
	tx, err := pool.BeginTx(ctx, options)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SET LOCAL lock_timeout='1s'; SET LOCAL statement_timeout='60s'"); err != nil {
		return result, err
	}
	var instances []string
	if err = tx.QueryRow(ctx, `SELECT coalesce(array_agg(instance_id::text),'{}') FROM organization.enterprise`).Scan(&instances); err != nil {
		return result, err
	}
	if !slices.Contains(instances, input.Instance) {
		return result, errors.New("maintenance instance does not exist")
	}
	for _, id := range instances {
		if !slices.Contains(input.OwnedInstances, id) {
			return result, errors.New("database contains an instance outside task ownership")
		}
	}
	if !input.DryRun {
		if err = maintenanceQuiescent(ctx, tx); err != nil {
			return result, err
		}
		if _, err = tx.Exec(ctx, `LOCK TABLE organization.bu_dictionary,organization.bu_history,organization.bu_read_segment,organization.bu_terminal,organization.department_dictionary,organization.department_history,organization.department_read_segment,organization.department_terminal IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return result, err
		}
		if err = maintenanceQuiescent(ctx, tx); err != nil {
			return result, err
		}
	}
	if result.Before, err = dictionaryGarbage(ctx, tx, input.Instance); err != nil {
		return result, err
	}
	if input.DryRun {
		result.After = result.Before
		result.Seconds = time.Since(started).Seconds()
		return result, nil
	}
	if result.ProtectedRoots, err = historyMaintenanceRoots(ctx, tx, input.Instance); err != nil {
		return result, err
	}
	for _, kind := range []history.Kind{history.BusinessUnit, history.Department} {
		if err = history.PruneDictionaries(ctx, tx, kind, input.Instance); err != nil {
			return result, err
		}
	}
	if result.After, err = dictionaryGarbage(ctx, tx, input.Instance); err != nil {
		return result, err
	}
	for i := range result.After {
		if result.After[i].Rows != 0 {
			return result, errors.New("unreferenced dictionary cleanup did not finish")
		}
		result.Removed += result.Before[i].Rows - result.After[i].Rows
	}
	after, err := historyMaintenanceRoots(ctx, tx, input.Instance)
	if err != nil {
		return result, err
	}
	if after != result.ProtectedRoots {
		return result, errors.New("history roots changed during dictionary maintenance")
	}
	if err = maintenanceQuiescent(ctx, tx); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, err
	}
	result.Seconds = time.Since(started).Seconds()
	return result, nil
}

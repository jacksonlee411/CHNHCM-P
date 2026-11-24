package storageverify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"chnhcm.local/chnhcm/internal/bootstrap"
	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

// Restore only the immutable, single-source tree fixture already produced by
// ordinary HTTP. This is an offline verification entry, never a product import
// route. The existing typed encoder, constraints and roundtrip verifier remain
// active for every object; no source identity or historical value is remapped.
func importFrozenTree(ctx context.Context, dir string) (any, error) {
	db, pool, err := openDatabase(ctx, dir)
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	var spec struct{ Facts, SHA256, EnterpriseName string }
	raw, err := os.ReadFile(filepath.Join(dir, "frozen-tree.json"))
	if err != nil || json.Unmarshal(raw, &spec) != nil || !filepath.IsAbs(spec.Facts) || spec.EnterpriseName == "" {
		return nil, errors.New("explicit private frozen-tree input required")
	}
	if err = externalDirectory(filepath.Dir(spec.Facts)); err != nil {
		return nil, err
	}
	file, err := os.Open(spec.Facts)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil || hex.EncodeToString(hash.Sum(nil)) != spec.SHA256 {
		return nil, errors.New("frozen-tree input digest differs")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(file)
	type envelope struct {
		Kind string
		Fact json.RawMessage
	}
	var first envelope
	if err = decoder.Decode(&first); err != nil || first.Kind != "BU" {
		return nil, errors.New("frozen tree must start with its one BU")
	}
	var bu bootstrap.BUHistoryFact
	if err = json.Unmarshal(first.Fact, &bu); err != nil || bu.Operation.Source.Number != 1 || bu.Operation.Source.Previous != "" {
		return nil, errors.New("frozen tree requires its original creation source")
	}
	if err = initializeFrozenTree(ctx, pool, bu, spec.EnterpriseName); err != nil {
		return nil, err
	}
	// Preserve extra ownership fields (including storage130's A role).
	configPath := filepath.Join(dir, "candidate-private.json")
	raw, err = os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var config map[string]any
	if err = json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	config["Instance"] = bu.Instance
	if err = writeJSON(configPath, config, 0600); err != nil {
		return nil, err
	}
	ownerOf := func(id, code, first, source string) owner {
		return owner{Instance: bu.Instance, Enterprise: bu.Enterprise, Code: code, NormalizedCode: strings.ToLower(code), First: first, Head: source, DepartmentID: id}
	}
	o := ownerOf("", bu.State.Timeline.Code, bu.State.Timeline.First, bu.Operation.Source.ID)
	o.BUID = bu.Object
	if err = writeBundle(ctx, pool, bundle{Kind: "bu", Owner: o, Frames: []json.RawMessage{first.Fact}}); err != nil {
		return nil, err
	}
	group, work := errgroup.WithContext(ctx)
	jobs := make(chan bundle, 8)
	var restored atomic.Int64
	for range 4 {
		group.Go(func() error {
			for b := range jobs {
				if err := writeBundle(work, pool, b); err != nil {
					return err
				}
				if n := restored.Add(1); n%1000 == 0 {
					fmt.Fprintln(os.Stderr, "restored frozen departments", n)
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(jobs)
		seen := map[string]bool{}
		for {
			var row envelope
			if err := decoder.Decode(&row); err == io.EOF {
				if len(seen) != 87343 {
					return errors.New("frozen tree department count differs")
				}
				return nil
			} else if err != nil {
				return err
			}
			var d bootstrap.DepartmentHistoryFact
			if row.Kind != "department" || json.Unmarshal(row.Fact, &d) != nil || d.Instance != bu.Instance || d.Enterprise != bu.Enterprise || d.Operation.Source.Subject != bu.Operation.Source.Subject || d.Operation.Source.Number != 1 || d.Operation.Source.Previous != "" || seen[d.Object] {
				return errors.New("frozen tree source identity differs")
			}
			seen[d.Object] = true
			b := bundle{Kind: "department", Owner: ownerOf(d.Object, d.State.Timeline.Code, d.State.Timeline.First, d.Operation.Source.ID), Frames: []json.RawMessage{row.Fact}}
			select {
			case jobs <- b:
			case <-work.Done():
				return work.Err()
			}
		}
	})
	if err = group.Wait(); err != nil {
		return nil, err
	}
	return map[string]any{"container": db.Container, "sources": restored.Load() + 1, "typedRoundtripVerified": true, "inputSHA256": spec.SHA256, "scope": "offline fixture restoration; not HTTP creation or capacity acceptance"}, nil
}

func initializeFrozenTree(ctx context.Context, pool *pgxpool.Pool, bu bootstrap.BUHistoryFact, name string) error {
	var occupied int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM organization.enterprise)+(SELECT count(*) FROM access.principal)+(SELECT count(*) FROM organization.business_unit)+(SELECT count(*) FROM organization.department)`).Scan(&occupied); err != nil {
		return err
	}
	if occupied != 0 {
		return errors.New("frozen tree restoration requires an empty owned database")
	}
	declaration, _ := json.Marshal([]string{bu.Instance, "LOCAL", name, foundation.DefaultEffectiveDate})
	digest := sha256.Sum256(declaration)
	if _, err := pool.Exec(ctx, `INSERT INTO organization.enterprise VALUES($1,$2,'LOCAL',$3,daterange($4::date,NULL,'[)'),$5)`, bu.Instance, bu.Enterprise, name, foundation.DefaultEffectiveDate, digest[:]); err != nil {
		return err
	}
	for _, login := range []string{"admin", "reader"} {
		// Establish the original source actor before the current initializer
		// creates enterprise history/receipts referring to that identity.
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		_, err = accessdb.Initialize(ctx, tx, bu.Instance, bu.Enterprise, login, "admin1234567", login == "reader")
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
	}
	// Reattach the freshly initialized admin to the source's exact actor before
	// any session/history exists. Preserve normal initialization and credentials;
	// this does not edit an existing environment or rewrite any business source.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM access.session)+(SELECT count(*) FROM access.security_event)`).Scan(&occupied); err != nil || occupied != 0 {
		return errors.New("fresh fixture accounts already used")
	}
	var admin string
	if err = tx.QueryRow(ctx, `SELECT principal_id::text FROM access.principal WHERE instance_id=$1 AND login='admin'`, bu.Instance).Scan(&admin); err != nil {
		return err
	}
	tables := []string{"access.subject_grant", "access.credential", "access.policy_guard", "access.principal"}
	rows := make(map[string][]byte, len(tables))
	for _, table := range tables {
		var raw []byte
		if err = tx.QueryRow(ctx, `WITH removed AS (DELETE FROM `+table+` WHERE instance_id=$1 AND principal_id=$2 RETURNING *) SELECT jsonb_agg(jsonb_set(to_jsonb(r),'{principal_id}',to_jsonb($3::text))) FROM removed r`, bu.Instance, admin, bu.Operation.Source.Subject).Scan(&raw); err != nil {
			return err
		}
		rows[table] = raw
	}
	for i := len(tables) - 1; i >= 0; i-- {
		table := tables[i]
		if _, err = tx.Exec(ctx, `INSERT INTO `+table+` SELECT * FROM jsonb_populate_recordset(NULL::`+table+`,$1::jsonb)`, rows[table]); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return bootstrap.Initialize(ctx, pool, bootstrap.InitInput{Instance: bu.Instance, EnterpriseCode: "LOCAL", EnterpriseName: name, Login: "admin", Password: "admin1234567"})
}

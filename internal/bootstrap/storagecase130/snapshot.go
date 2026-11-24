package storagecase130

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

type tableSnapshot struct {
	Schema, Name, File, SHA256 string
	Columns, Keys              []string
	Rows                       int64
}

type snapshot struct {
	Instance, Model string
	CapturedAt      time.Time
	Tables          []tableSnapshot
}

func tableName(t tableSnapshot) string { return pgx.Identifier{t.Schema, t.Name}.Sanitize() }
func columns(names []string) string {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = pgx.Identifier{name}.Sanitize()
	}
	return strings.Join(out, ",")
}

func exportSource(ctx context.Context, dir string) (any, error) {
	var source database
	if err := readPrivate(filepath.Join(dir, "source-private.json"), &source); err != nil {
		return nil, err
	}
	labelFormat := "{{index .Config.Labels \"io.chnhcm.profile\"}}/{{index .Config.Labels \"io.chnhcm.temporary\"}}"
	wantedLabel, wantedDatabase := "m02-current-vanke-b/false", "/chnhcm"
	if source.Role == "A" {
		container, volume, task, e := resources("A")
		if e != nil || source.Container != container || source.Volume != volume {
			return nil, errors.New("foreign frozen baseline source")
		}
		labelFormat = "{{index .Config.Labels \"io.chnhcm.task\"}}/{{index .Config.Labels \"io.chnhcm.parent-task\"}}/{{index .Config.Labels \"io.chnhcm.temporary\"}}"
		wantedLabel, wantedDatabase = task+"/storage130/true", "/storage126"
	}
	labels, err := docker("inspect", "--format", labelFormat, source.Container)
	if err != nil || strings.TrimSpace(string(labels)) != wantedLabel {
		return nil, errors.New("source is not the approved retained profile or owned frozen baseline")
	}
	port, err := docker("port", source.Container, "5432/tcp")
	u, parseErr := url.Parse(source.AdminDSN)
	if err != nil || parseErr != nil || u.Host != strings.TrimSpace(string(port)) || u.Hostname() != "127.0.0.1" || u.Path != wantedDatabase {
		return nil, errors.New("read-only source connection differs from its owned container")
	}
	pool, err := pgxpool.New(ctx, source.AdminDSN)
	if err != nil {
		return nil, errors.New("invalid private source connection")
	}
	defer pool.Close()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, errors.New("read-only source snapshot unavailable")
	}
	defer tx.Rollback(context.Background())
	manifest := snapshot{Instance: source.Instance, CapturedAt: time.Now()}
	if err = tx.QueryRow(ctx, `SELECT contract FROM organization.storage_contract`).Scan(&manifest.Model); err != nil || manifest.Model != "m02-compact-3" {
		return nil, errors.New("source snapshot must be the explicitly approved model-3 case")
	}
	var owner bool
	if err = tx.QueryRow(ctx, `SELECT count(*)=1 AND bool_and(instance_id=$1::uuid) FROM organization.enterprise`, source.Instance).Scan(&owner); err != nil || !owner {
		return nil, errors.New("source instance differs from the approved case")
	}
	rows, err := tx.Query(ctx, `SELECT n.nspname,c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access','verification') AND c.relname<>'storage_contract' ORDER BY n.nspname,c.relname`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var t tableSnapshot
		if err = rows.Scan(&t.Schema, &t.Name); err != nil {
			rows.Close()
			return nil, err
		}
		manifest.Tables = append(manifest.Tables, t)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range manifest.Tables {
		t := &manifest.Tables[i]
		qualified := t.Schema + "." + t.Name
		if err = tx.QueryRow(ctx, `SELECT array_agg(attname::text ORDER BY attnum) FROM pg_attribute WHERE attrelid=$1::regclass AND attnum>0 AND NOT attisdropped AND attgenerated='' AND attname NOT IN ('head_xid','written_xid')`, qualified).Scan(&t.Columns); err != nil {
			return nil, err
		}
		if err = tx.QueryRow(ctx, `SELECT array_agg(a.attname::text ORDER BY k.ord) FROM pg_index i CROSS JOIN LATERAL unnest(i.indkey) WITH ORDINALITY k(num,ord) JOIN pg_attribute a ON a.attrelid=i.indrelid AND a.attnum=k.num WHERE i.indrelid=$1::regclass AND i.indisprimary AND k.ord<=i.indnkeyatts`, qualified).Scan(&t.Keys); err != nil || len(t.Keys) == 0 {
			return nil, errors.New("snapshot table lacks a stable primary directory")
		}
		t.File = t.Schema + "." + t.Name + ".jsonl.gz"
		f, err := os.OpenFile(filepath.Join(dir, t.File), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, err
		}
		z := gzip.NewWriter(f)
		h := sha256.New()
		selects := make([]string, len(t.Columns))
		for n, column := range t.Columns {
			selects[n] = "t." + pgx.Identifier{column}.Sanitize()
		}
		query := `SELECT row_to_json(v)::jsonb FROM (SELECT ` + strings.Join(selects, ",") + " FROM " + tableName(*t) + " t ORDER BY " + columns(t.Keys) + ") v"
		data, err := tx.Query(ctx, query)
		if err != nil {
			z.Close()
			f.Close()
			return nil, err
		}
		for data.Next() {
			var raw []byte
			if err = data.Scan(&raw); err != nil {
				break
			}
			raw = append(raw, '\n')
			if _, err = z.Write(raw); err != nil {
				break
			}
			_, _ = h.Write(raw)
			t.Rows++
		}
		if err == nil {
			err = data.Err()
		}
		data.Close()
		closeZ, closeFile := z.Close(), f.Close()
		if err != nil {
			return nil, err
		}
		if closeZ != nil {
			return nil, closeZ
		}
		if closeFile != nil {
			return nil, closeFile
		}
		t.SHA256 = hex.EncodeToString(h.Sum(nil))
	}
	if err = writePrivate(filepath.Join(dir, "snapshot.json"), manifest); err != nil {
		return nil, err
	}
	return map[string]any{"capturedAt": manifest.CapturedAt, "model": manifest.Model, "tables": manifest.Tables, "readOnly": true}, nil
}

type rowStream struct {
	f       *os.File
	z       *gzip.Reader
	scanner *bufio.Scanner
	hash    hash.Hash
	table   tableSnapshot
	count   int64
}

func stream(dir string, t tableSnapshot) (*rowStream, error) {
	if filepath.Base(t.File) != t.File || t.File != t.Schema+"."+t.Name+".jsonl.gz" {
		return nil, errors.New("invalid snapshot member")
	}
	f, err := os.Open(filepath.Join(dir, t.File))
	if err != nil {
		return nil, err
	}
	z, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	s := bufio.NewScanner(z)
	s.Buffer(make([]byte, 65536), 64<<20)
	return &rowStream{f: f, z: z, scanner: s, hash: sha256.New(), table: t}, nil
}

func (s *rowStream) next() (json.RawMessage, error) {
	if s.scanner.Scan() {
		raw := slices.Clone(s.scanner.Bytes())
		_, _ = s.hash.Write(raw)
		_, _ = s.hash.Write([]byte{'\n'})
		s.count++
		return raw, nil
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	if s.count != s.table.Rows || hex.EncodeToString(s.hash.Sum(nil)) != s.table.SHA256 {
		return nil, errors.New("snapshot row count or content checksum differs")
	}
	return nil, io.EOF
}

func (s *rowStream) close() { s.z.Close(); s.f.Close() }

func insertRows(ctx context.Context, tx pgx.Tx, t tableSnapshot, rows []json.RawMessage) error {
	if len(rows) == 0 {
		return nil
	}
	body, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	if t.Schema == "access" && t.Name == "subject_grant" {
		return bootstrap.RestoreGrantArchive(ctx, tx, body)
	}
	_, err = tx.Exec(ctx, "INSERT INTO "+tableName(t)+" ("+columns(t.Columns)+") OVERRIDING SYSTEM VALUE SELECT "+columns(t.Columns)+" FROM jsonb_populate_recordset(NULL::"+tableName(t)+",$1::jsonb)", body)
	return err
}

func importTable(ctx context.Context, pool *pgxpool.Pool, dir string, t tableSnapshot) error {
	s, err := stream(dir, t)
	if err != nil {
		return err
	}
	defer s.close()
	for {
		batch := make([]json.RawMessage, 0, 256)
		for len(batch) < cap(batch) {
			raw, err := s.next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			batch = append(batch, raw)
		}
		if len(batch) == 0 {
			return nil
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		err = insertRows(ctx, tx, t, batch)
		if err == nil {
			err = tx.Commit(ctx)
		}
		_ = tx.Rollback(context.Background())
		if err != nil {
			return fmt.Errorf("import %s failed: %w", t.Name, err)
		}
	}
}

type sourceDirectory struct {
	Instance    string   `json:"instance_id"`
	Object      string   `json:"object_id"`
	IDs         []string `json:"ids"`
	Subjects    []int    `json:"subject_refs"`
	Submissions []int    `json:"submission_refs"`
}

func lockSources(ctx context.Context, tx pgx.Tx, namespace string, histories []json.RawMessage) error {
	actors, submissions := []string{}, []string{}
	instance := ""
	for _, raw := range histories {
		var d sourceDirectory
		if err := json.Unmarshal(raw, &d); err != nil {
			return err
		}
		instance = d.Instance
		if len(d.Subjects) != len(d.Submissions) {
			return errors.New("invalid source identity directory")
		}
		for i, a := range d.Subjects {
			s := d.Submissions[i]
			if a < 1 || a > len(d.IDs) || s < 1 || s > len(d.IDs) {
				return errors.New("source identity reference out of range")
			}
			actors = append(actors, d.IDs[a-1])
			submissions = append(submissions, d.IDs[s-1])
		}
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(organization.submission_key($1,$2,p.actor,p.submission)) FROM unnest($3::uuid[],$4::uuid[]) p(actor,submission) ORDER BY p.actor,p.submission`, namespace, instance, actors, submissions)
	return err
}

func importObjects(ctx context.Context, pool *pgxpool.Pool, dir, prefix string, owner, historyTable tableSnapshot, relations map[string]json.RawMessage, relationTable tableSnapshot) error {
	owners, err := stream(dir, owner)
	if err != nil {
		return err
	}
	defer owners.close()
	histories, err := stream(dir, historyTable)
	if err != nil {
		return err
	}
	defer histories.close()
	type object struct {
		owner     json.RawMessage
		histories []json.RawMessage
		relation  json.RawMessage
	}
	group, work := errgroup.WithContext(ctx)
	jobs := make(chan object, 8)
	for worker := 0; worker < 4; worker++ {
		group.Go(func() error {
			for item := range jobs {
				if work.Err() != nil {
					return work.Err()
				}
				err := func() error {
					tx, err := pool.Begin(work)
					if err != nil {
						return err
					}
					defer tx.Rollback(context.Background())
					if err = lockSources(work, tx, prefix, item.histories); err != nil {
						return err
					}
					if err = insertRows(work, tx, owner, []json.RawMessage{item.owner}); err != nil {
						return err
					}
					for _, h := range item.histories {
						if err = insertRows(work, tx, historyTable, []json.RawMessage{h}); err != nil {
							return err
						}
					}
					if item.relation != nil {
						if err = insertRows(work, tx, relationTable, []json.RawMessage{item.relation}); err != nil {
							return err
						}
					}
					return tx.Commit(work)
				}()
				if err != nil {
					return fmt.Errorf("offline %s object import: %w", prefix, err)
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(jobs)
		pending, historyErr := histories.next()
		for {
			raw, err := owners.next()
			if err == io.EOF {
				if historyErr != io.EOF {
					return errors.New("snapshot has unowned history rows")
				}
				return nil
			}
			if err != nil {
				return err
			}
			var identity struct {
				BU         string `json:"bu_id"`
				Department string `json:"department_id"`
			}
			if err = json.Unmarshal(raw, &identity); err != nil {
				return err
			}
			id := identity.BU
			if prefix == "department" {
				id = identity.Department
			}
			item := object{owner: raw, relation: relations[id]}
			for historyErr == nil {
				var d sourceDirectory
				if err = json.Unmarshal(pending, &d); err != nil {
					return err
				}
				if d.Object != id {
					break
				}
				item.histories = append(item.histories, pending)
				pending, historyErr = histories.next()
			}
			if historyErr != nil && historyErr != io.EOF {
				return historyErr
			}
			if len(item.histories) == 0 {
				return errors.New("object lacks its exact history")
			}
			select {
			case jobs <- item:
			case <-work.Done():
				return work.Err()
			}
		}
	})
	return group.Wait()
}

func importTerminals(ctx context.Context, pool *pgxpool.Pool, dir, prefix string, t tableSnapshot) error {
	s, err := stream(dir, t)
	if err != nil {
		return err
	}
	defer s.close()
	for {
		raw, err := s.next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var identity struct {
			Instance   string `json:"instance_id"`
			Subject    string `json:"subject_id"`
			Submission string `json:"submission_id"`
		}
		if err = json.Unmarshal(raw, &identity); err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(organization.submission_key($1,$2,$3,$4))`, prefix, identity.Instance, identity.Subject, identity.Submission)
		if err == nil {
			err = insertRows(ctx, tx, t, []json.RawMessage{raw})
		}
		if err == nil {
			err = tx.Commit(ctx)
		}
		_ = tx.Rollback(context.Background())
		if err != nil {
			return err
		}
	}
}

func importSnapshot(ctx context.Context, dir string, db database, pool *pgxpool.Pool) (any, error) {
	var manifest snapshot
	if err := readPrivate(filepath.Join(dir, "snapshot.json"), &manifest); err != nil {
		return nil, err
	}
	if manifest.Model != "m02-compact-3" || manifest.Instance == "" {
		return nil, errors.New("snapshot source format is not the approved model-3 case")
	}
	var occupied int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM organization.enterprise)+(SELECT count(*) FROM access.principal)`).Scan(&occupied); err != nil || occupied != 0 {
		return nil, errors.New("snapshot import requires an empty task database")
	}
	tables := map[string]tableSnapshot{}
	for _, t := range manifest.Tables {
		tables[t.Schema+"."+t.Name] = t
	}
	for _, name := range []string{"organization.enterprise", "access.principal", "access.credential", "access.policy_guard", "access.subject_grant", "access.session", "access.security_event", "organization.bu_dictionary", "organization.bu_shape", "organization.department_dictionary", "organization.department_shape"} {
		t, ok := tables[name]
		if !ok {
			return nil, fmt.Errorf("snapshot lacks %s", name)
		}
		if err := importTable(ctx, pool, dir, t); err != nil {
			return nil, err
		}
	}
	// Restored internal IDs remain unchanged; new shape insertion must not
	// collide with the imported IDs. Never rewind an existing allocator.
	for _, name := range []string{"bu_dictionary", "bu_shape", "department_dictionary", "department_shape"} {
		if _, err := pool.Exec(ctx, `SELECT setval(pg_get_serial_sequence('organization.`+name+`','id'),greatest(coalesce((SELECT max(id) FROM organization.`+name+`),1),1),true)`); err != nil {
			return nil, err
		}
	}
	relations := map[string]json.RawMessage{}
	relationTable := tables["organization.department_hierarchy_relation"]
	s, err := stream(dir, relationTable)
	if err != nil {
		return nil, err
	}
	for {
		raw, nextErr := s.next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			s.close()
			return nil, nextErr
		}
		var identity struct {
			Department string `json:"department_id"`
		}
		if err = json.Unmarshal(raw, &identity); err != nil {
			s.close()
			return nil, err
		}
		relations[identity.Department] = raw
	}
	s.close()
	for _, g := range []struct{ prefix, owner string }{{"bu", "business_unit"}, {"department", "department"}} {
		if err = importObjects(ctx, pool, dir, g.prefix, tables["organization."+g.owner], tables["organization."+g.prefix+"_history"], relationsFor(g.prefix, relations), relationTable); err != nil {
			return nil, err
		}
		if err = importTerminals(ctx, pool, dir, g.prefix, tables["organization."+g.prefix+"_terminal"]); err != nil {
			return nil, err
		}
	}
	for _, t := range manifest.Tables {
		if t.Schema == "verification" {
			if _, err = pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS verification"); err != nil {
				return nil, err
			}
			// Verification schemas require an explicit identical DDL supplied by
			// their dataset owner; never drop the raw source to improve size.
			if err = importTable(ctx, pool, dir, t); err != nil {
				return nil, err
			}
		}
	}
	if _, err = bootstrap.RebuildCurrentReadIndexes(ctx, pool, manifest.Instance); err != nil {
		return nil, err
	}
	db.Instance = manifest.Instance
	if err = writePrivate(filepath.Join(dir, "candidate-private.json"), db); err != nil {
		return nil, err
	}
	return map[string]any{"imported": true, "role": db.Role, "sourceModel": manifest.Model, "businessIdentitiesPreserved": true, "constraintsDisabled": false}, nil
}

func relationsFor(prefix string, values map[string]json.RawMessage) map[string]json.RawMessage {
	if prefix == "department" {
		return values
	}
	return nil
}

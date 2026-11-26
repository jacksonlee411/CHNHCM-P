//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"io/fs"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The input is a private dump from the previous product binary. Its synthetic
// login belongs to the fixture. No database credentials enter the worktree.
func TestPersonnelUpgradeFromBackup(t *testing.T) {
	dir := os.Getenv("CHNHCM_PERSONNEL_BACKUP_DIR")
	if dir == "" {
		t.Skip("explicit pre-T01 PostgreSQL backup")
	}
	raw, e := os.ReadFile(filepath.Join(dir, "fixture.json"))
	must(t, e)
	var seed struct{ Instance, Password, BU, Position string }
	must(t, json.Unmarshal(raw, &seed))
	f := fresh(t)
	container := f.container.GetContainerID()
	restore := func(name, path string) (*pgxpool.Pool, string) {
		_, e := f.admin.Exec(f.ctx, "create database "+name)
		must(t, e)
		input, e := os.Open(path)
		must(t, e)
		defer input.Close()
		cmd := exec.Command("docker", "exec", "-i", container, "pg_restore", "--exit-on-error", "-U", "postgres", "-d", name)
		cmd.Stdin = input
		must(t, cmd.Run())
		cfg := f.admin.Config()
		cfg.ConnConfig.Database = name
		admin, e := pgxpool.NewWithConfig(f.ctx, cfg)
		must(t, e)
		t.Cleanup(admin.Close)
		// ConnString returns the originally parsed URL. Replace only its database.
		u, e := url.Parse(f.runtime.Pool.Config().ConnString())
		must(t, e)
		u.Path = "/" + name
		return admin, u.String()
	}
	serve := func(admin *pgxpool.Pool, dsn string) *fixture {
		r, e := bootstrap.Open(f.ctx, dsn, seed.Instance)
		must(t, e)
		t.Cleanup(r.Close)
		server := httptest.NewUnstartedServer(nil)
		a := bootstrap.API{Runtime: r, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}
		server.Config.Handler = a.Handler()
		server.Start()
		t.Cleanup(server.Close)
		return &fixture{t: t, ctx: f.ctx, admin: admin, instance: seed.Instance, server: server, runtime: r}
	}
	admin, dsn := restore("before_personnel", filepath.Join(dir, "before.dump"))
	var baseline int
	must(t, admin.QueryRow(f.ctx, "select max(version_id) from public.goose_db_version where is_applied").Scan(&baseline))
	if baseline != 25 {
		t.Fatal("expected a pre-T01 backup")
	}
	if r, e := bootstrap.Open(f.ctx, dsn, seed.Instance); e == nil {
		r.Close()
		t.Fatal("admitted unupgraded database")
	}
	for range 2 {
		must(t, bootstrap.Migrate(f.ctx, admin, migrations.Files))
	}
	g := serve(admin, dsn)
	hr := g.login("admin", seed.Password)
	hr.call("GET", "/api/v1/business-units/"+seed.BU, nil, "", 200)
	hr.call("GET", "/api/v1/positions/"+seed.Position, nil, "", 200)
	hr.call("GET", "/api/v1/persons", nil, "", 403)
	must(t, bootstrap.Initialize(f.ctx, admin, bootstrap.InitInput{Instance: seed.Instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "1900-01-01", Login: "admin", Password: seed.Password, PersonnelProfile: "registrar"}))
	type M = map[string]any
	const base = "/api/v1/persons"
	register := func(name, ref, document string) M {
		return hr.call("POST", base, M{"submissionId": foundation.NewID(), "name": name, "evidence": M{"kind": "form", "reference": ref}, "primaryDocument": M{"kind": "passport", "number": document}}, "", 200)
	}
	v := register("升级样本", "BACK-1", "TEST-RESTORE-01")
	id := v["id"].(string)
	key := foundation.NewID()
	hr.call("POST", base+"/"+id+"/renumber", M{"submissionId": key, "expectedSourceVersion": v["sourceVersion"], "oldNumber": "P00000001", "newNumber": "R-01", "reason": "migration", "decisionRef": "D-01"}, "", 200)
	expired := register("保全样本", "HOLD-1", "TEST-RESTORE-02")
	_, e = admin.Exec(f.ctx, "update personnel.person set ordinary_until=now()-interval '2 days',evidence_until=now()-interval '1 day',material_until=now()-interval '2 days',material_evidence_until=now()-interval '1 day' where person_id=$1", expired["id"])
	must(t, e)
	_, e = admin.Exec(f.ctx, "insert into personnel.hold values($1,$2,$3,'evidence',now()+interval '1 day','H-1')", seed.Instance, expired["id"], expired["sourceVersion"])
	must(t, e)
	dump, e := os.CreateTemp(t.TempDir(), "after-*.dump")
	must(t, e)
	cmd := exec.Command("docker", "exec", container, "pg_dump", "-U", "postgres", "-Fc", "before_personnel")
	cmd.Stdout = dump
	must(t, cmd.Run())
	must(t, dump.Close())
	after, newDSN := restore("after_personnel", dump.Name())
	must(t, bootstrap.Migrate(f.ctx, after, migrations.Files))
	restored := serve(after, newDSN).login("admin", seed.Password)
	restored.call("GET", "/api/v1/business-units/"+seed.BU, nil, "", 200)
	restored.call("GET", "/api/v1/positions/"+seed.Position, nil, "", 200)
	if restored.call("GET", base+"/"+id+"/sources/"+v["sourceVersion"].(string), nil, "", 200)["number"] != "P00000001" || restored.call("POST", base+"/submissions/"+key+"/resolve", nil, "", 200)["id"] != id {
		t.Fatal("lost source or receipt")
	}
	if restored.call("GET", base+"/"+id+"?view=fullDocument", nil, "", 200)["primaryDocument"].(M)["number"] != "TEST-RESTORE-01" {
		t.Fatal("restored file lost its document")
	}
	restored.call("POST", base, M{"submissionId": foundation.NewID(), "name": "异名重录", "primaryDocument": M{"kind": "other", "number": "test-restore-01"}}, "", 409)
	restored.call("GET", base+"/"+expired["id"].(string), nil, "", 404)
	var intact bool
	must(t, after.QueryRow(f.ctx, "select exists(select from personnel.number where number='P00000001' and person_id=$1) and not exists(select from personnel.identity_payload where person_id=$2) and exists(select from personnel.evidence_payload where person_id=$2 and reference='HOLD-1')", id, expired["id"]).Scan(&intact))
	if !intact {
		t.Fatal("reservation, disposal or hold changed on restore")
	}
}

// Exercise a protected 00026 backup without opening its database for writes.
func TestPersonnelPrimaryDocumentUpgrade(t *testing.T) {
	path := os.Getenv("CHNHCM_PERSONNEL_V26_DUMP")
	if path == "" {
		t.Skip("requires a private copy of a 00026 backup")
	}
	f := fresh(t)
	_, err := f.admin.Exec(f.ctx, "create database personnel_v26_copy")
	must(t, err)
	dump, err := os.Open(path)
	must(t, err)
	defer dump.Close()
	cmd := exec.Command("docker", "exec", "-i", f.container.GetContainerID(), "pg_restore", "--exit-on-error", "-U", "postgres", "-d", "personnel_v26_copy")
	cmd.Stdin = dump
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("restore: %v %s", err, output)
	}
	cfg := f.admin.Config()
	cfg.ConnConfig.Database = "personnel_v26_copy"
	admin, err := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, err)
	defer admin.Close()
	fingerprint := func() map[string]string {
		tables := []string{"person", "number", "source", "identity_payload", "evidence_payload", "contact_payload", "counter", "material", "outbox", "terminal", "hold", "disposition"}
		out := map[string]string{}
		for _, table := range tables {
			var state string
			fields := "to_jsonb(v)"
			if table == "identity_payload" {
				fields += " - 'document_kind' - 'document_number' - 'current_document' - 'basic' - 'nationality_adoption'"
			}
			if table == "contact_payload" {
				fields += " - 'purpose' - 'is_primary'"
			}
			err := admin.QueryRow(f.ctx, "select count(*)||'/'||md5(coalesce(string_agg(row_value,'|' order by row_value),'')) from (select ("+fields+")::text row_value from personnel."+table+" v) snapshot").Scan(&state)
			must(t, err)
			out[table] = state
		}
		return out
	}
	var ledger int
	must(t, admin.QueryRow(f.ctx, "select max(version_id) from public.goose_db_version where is_applied").Scan(&ledger))
	if ledger != 26 {
		t.Fatal("expected migration 00026")
	}
	var instance string
	must(t, admin.QueryRow(f.ctx, "select instance_id from personnel.person limit 1").Scan(&instance))
	u, err := url.Parse(f.runtime.Pool.Config().ConnString())
	must(t, err)
	u.Path = "/personnel_v26_copy"
	if runtime, openErr := bootstrap.Open(f.ctx, u.String(), instance); openErr == nil {
		runtime.Close()
		t.Fatal("new service admitted migration 00026")
	}
	before := fingerprint()
	for range 2 {
		must(t, bootstrap.Migrate(f.ctx, admin, migrations.Files))
	}
	if !reflect.DeepEqual(before, fingerprint()) {
		t.Fatal("upgrade changed an existing personnel fact")
	}
	var people, numbers, sources, materials, receipts, documents int
	must(t, admin.QueryRow(f.ctx, `select (select count(*) from personnel.person),(select count(*) from personnel.number),(select count(*) from personnel.source),(select count(*) from personnel.material),(select count(*) from personnel.terminal),(select count(*) from personnel.document_payload)`).Scan(&people, &numbers, &sources, &materials, &receipts, &documents))
	if people == 0 || documents != 0 {
		t.Fatal("empty backup or invented document numbers")
	}
	t.Logf("升级前后原记录一致：%d份档案、%d个现旧编号、%d次资料记录、%d个单据引用、%d笔办理结果。原档证件号码均为空。", people, numbers, sources, materials, receipts)
	var id, actor, submit, version, name, kind, ref string
	must(t, admin.QueryRow(f.ctx, `select m.instance_id,m.person_id,s.operator_name,t.submission_id,s.version,n.name,e.kind,e.reference
 from personnel.material m join personnel.source s on (s.instance_id,s.person_id,s.version)=(m.instance_id,m.person_id,m.source_version)
 join personnel.terminal t on (t.instance_id,t.person_id,t.version)=(s.instance_id,s.person_id,s.version)
 join personnel.identity_payload n on (n.instance_id,n.person_id,n.version)=(s.instance_id,s.person_id,s.version)
 join personnel.evidence_payload e on (e.instance_id,e.person_id,e.version)=(s.instance_id,s.person_id,s.version)
 where s.action='register' and m.registration_hash is not null and not exists(select from personnel.contact_payload c where (c.instance_id,c.person_id,c.version)=(s.instance_id,s.person_id,s.version)) limit 1`).Scan(&instance, &id, &actor, &submit, &version, &name, &kind, &ref))
	r, err := bootstrap.Open(f.ctx, u.String(), instance)
	must(t, err)
	defer r.Close()
	server := httptest.NewUnstartedServer(nil)
	api := bootstrap.API{Runtime: r, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}
	server.Config.Handler = api.Handler()
	server.Start()
	defer server.Close()
	g := fixture{t: t, ctx: f.ctx, admin: admin, instance: instance, server: server, runtime: r}
	b := g.login(actor, "admin1234567")
	const route = "/api/v1/persons"
	type M = map[string]any
	rows, err := admin.Query(f.ctx, `select person_id,number from personnel.person where instance_id=$1 and ordinary_until>clock_timestamp() order by number`, instance)
	must(t, err)
	for rows.Next() {
		var personID, employeeNumber string
		must(t, rows.Scan(&personID, &employeeNumber))
		file := b.call("GET", route+"/"+personID, nil, "", 200)
		if file["number"] != employeeNumber || file["primaryDocument"] != nil || file["evidence"] != nil {
			t.Fatal("old file or ordinary projection changed")
		}
	}
	must(t, rows.Err())
	rows.Close()
	if b.call("POST", route+"/submissions/"+submit+"/resolve", nil, "", 200)["sourceVersion"] != version {
		t.Fatal("lost original receipt")
	}
	if actual := b.call("POST", route, M{"name": name, "submissionId": foundation.NewID(), "evidence": M{"kind": kind, "reference": ref}}, "", 200); actual["id"] != id || actual["sourceVersion"] != version || actual["outcome"] != "accepted" || actual["originalNumber"] == nil {
		t.Fatal("lost original form")
	}
	b.call("POST", route, M{"name": "补录权限样本", "submissionId": foundation.NewID(), "primaryDocument": M{"kind": "other", "number": "SYNTHETIC-COPY"}}, "", 403)
	// Only the fixture authority grants this clerk access to the document on this one file.
	_, err = admin.Exec(f.ctx, `insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids)
 select t.instance_id,t.actor,g.action,g.purpose,g.fields,array[t.person_id] from personnel.terminal t
 cross join (values ('read','directory',array['name','number','document']),('correct','maintenance',array['document'])) g(action,purpose,fields)
 where t.instance_id=$1 and t.submission_id=$2 and t.person_id=$3`, instance, submit, id)
	must(t, err)
	original := b.call("GET", route+"/"+id, nil, "", 200)
	b.call("POST", route+"/"+id+"/corrections", M{"submissionId": foundation.NewID(), "expectedSourceVersion": original["sourceVersion"], "primaryDocument": M{"kind": "passport", "number": "TEST-ARCHIVE-COPY"}}, "", 200)
	supplemented := b.call("GET", route+"/"+id+"?view=fullDocument", nil, "", 200)
	if supplemented["number"] != original["number"] || supplemented["primaryDocument"].(M)["number"] != "TEST-ARCHIVE-COPY" {
		t.Fatal("supplementing an existing file changed its number")
	}
	if old := b.call("GET", route+"/"+id+"/sources/"+version, nil, "", 200); old["primaryDocument"] != nil || old["evidence"] == nil {
		t.Fatal("supplementing a file changed its registration source")
	}

}

// Restore the manual-acceptance database into a separate PostgreSQL instance.
// Some v27 backups have no document yet. Exercise the two historical values on
// the copy only, leaving the supplied backup and the original customer untouched.
func TestPersonnelProfileV27Backup(t *testing.T) {
	path := os.Getenv("CHNHCM_PERSONNEL_V27_DUMP")
	if path == "" {
		t.Skip("requires private v27 backup")
	}
	f := fresh(t)
	_, err := f.admin.Exec(f.ctx, "create database profile_v27_copy")
	must(t, err)
	dump, err := os.Open(path)
	must(t, err)
	defer dump.Close()
	cmd := exec.Command("docker", "exec", "-i", f.container.GetContainerID(), "pg_restore", "--exit-on-error", "-U", "postgres", "-d", "profile_v27_copy")
	cmd.Stdin = dump
	must(t, cmd.Run())
	cfg := f.admin.Config()
	cfg.ConnConfig.Database = "profile_v27_copy"
	db, err := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, err)
	defer db.Close()
	var instance, id, head, previous string
	must(t, db.QueryRow(f.ctx, `select p.instance_id,p.person_id,p.head,s.previous from personnel.person p join personnel.source s on (p.instance_id,p.person_id,p.head)=(s.instance_id,s.person_id,s.version) where s.previous is not null limit 1`).Scan(&instance, &id, &head, &previous))
	_, err = db.Exec(f.ctx, `update personnel.identity_payload set document_kind='passport',document_number=case when version=$2 then 'SYNTH-OLD-27' else 'SYNTH-NEW-27' end,current_document=(version=$3) where person_id=$1 and version in ($2,$3)`, id, previous, head)
	must(t, err)
	fingerprint := func() map[string]string {
		out := map[string]string{}
		for _, table := range []string{"person", "number", "counter", "source", "material", "evidence_payload", "terminal", "outbox", "hold", "disposition", "identity_payload", "contact_payload"} {
			var state string
			query := "select md5(coalesce(string_agg(row_value,'|' order by row_value),'')) from (select (to_jsonb(v)-'basic'-'nationality_adoption'-'document_kind'-'document_number'-'current_document'-'purpose'-'is_primary')::text row_value from personnel." + table + " v) rows"
			must(t, db.QueryRow(f.ctx, query).Scan(&state))
			out[table] = state
		}
		return out
	}
	before := fingerprint()
	_, err = db.Exec(f.ctx, "create table personnel.document_payload(test_obstacle int)")
	must(t, err)
	if bootstrap.Migrate(f.ctx, db, migrations.Files) == nil {
		t.Fatal("migration accepted conflicting relation")
	}
	var ledger int
	var partial bool
	must(t, db.QueryRow(f.ctx, `select max(version_id),exists(select from pg_attribute where attrelid='personnel.identity_payload'::regclass and attname='basic') from goose_db_version where is_applied`).Scan(&ledger, &partial))
	if ledger != 30 || partial || !reflect.DeepEqual(before, fingerprint()) {
		t.Fatalf("failed migration left changes: version=%d", ledger)
	}
	_, err = db.Exec(f.ctx, "drop table personnel.document_payload")
	must(t, err)
	for range 2 {
		must(t, bootstrap.Migrate(f.ctx, db, migrations.Files))
	}
	if !reflect.DeepEqual(before, fingerprint()) {
		t.Fatal("existing facts changed on retry")
	}
	var count int
	must(t, db.QueryRow(f.ctx, `select count(*) from personnel.document_payload where person_id=$1 and is_primary and (version=$2 and number='SYNTH-OLD-27' and not occupied or version=$3 and number='SYNTH-NEW-27' and occupied)`, id, previous, head).Scan(&count))
	if count != 2 {
		t.Fatal("lost old document or present ownership")
	}
	u, err := url.Parse(f.runtime.Pool.Config().ConnString())
	must(t, err)
	u.Path = "/profile_v27_copy"
	runtime, err := bootstrap.Open(f.ctx, u.String(), instance)
	must(t, err)
	defer runtime.Close()
	server := httptest.NewUnstartedServer(nil)
	api := bootstrap.API{Runtime: runtime, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}
	server.Config.Handler = api.Handler()
	server.Start()
	defer server.Close()
	g := fixture{t: t, ctx: f.ctx, admin: db, instance: instance, server: server, runtime: runtime}
	hr := g.login("admin", "admin1234567")
	root := "/api/v1/persons/" + id
	original := hr.call("GET", root+"/sources/"+previous+"?view=fullDocument", nil, "", 200)
	if original["primaryDocument"] != nil {
		t.Fatal("old source grant broadened by upgrade")
	}
	_, err = db.Exec(f.ctx, `insert into access.person_grant(instance_id,principal_id,action,purpose,fields,person_ids) select instance_id,principal_id,'history.read','history',array['name','number','document'],array[$2::uuid] from access.principal where instance_id=$1 and login='admin'`, instance, id)
	must(t, err)
	for suffix, number := range map[string]string{"?view=fullDocument": "SYNTH-NEW-27", "/sources/" + previous + "?view=fullDocument": "SYNTH-OLD-27"} {
		d := hr.call("GET", root+suffix, nil, "", 200)
		if d["primaryDocument"].(map[string]any)["number"] != number || d["documents"] != nil || d["basic"] != nil {
			t.Fatal("incorrect document projection or inherited grants")
		}
	}
}

func TestPersonnelMainlinePrefixes(t *testing.T) {
	for _, version := range []int{26, 27, 28} {
		t.Run(strconv.Itoa(version), func(t *testing.T) {
			db := newStaffingMigrationPool(t)
			ctx := t.Context()
			files := fstest.MapFS{}
			err := fs.WalkDir(migrations.Files, ".", func(path string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				if !strings.Contains(path, "/") {
					v, _ := strconv.Atoi(path[:5])
					if v > version {
						return nil
					}
				}
				data, err := migrations.Files.ReadFile(path)
				if err == nil {
					files[path] = &fstest.MapFile{Data: data}
				}
				return err
			})
			must(t, err)
			must(t, bootstrap.Migrate(ctx, db, files))
			instance, enterprise, root, source := foundation.NewID(), foundation.NewID(), foundation.NewID(), foundation.NewID()
			_, err = db.Exec(ctx, "insert into staffing.position_hierarchy values($1,$2,$3)", instance, enterprise, root)
			must(t, err)
			_, err = db.Exec(ctx, "insert into staffing.position_hierarchy_version values($1,$2,$3,1,'{}',now(),$4)", instance, root, source, foundation.NewID())
			must(t, err)
			_, err = db.Exec(ctx, "insert into staffing.position_hierarchy_outbox values($1,$2,$3,'{}')", instance, root, source)
			must(t, err)
			check := func() string {
				var facts string
				must(t, db.QueryRow(ctx, "select row_to_json(v)::text||e.event::text from staffing.position_hierarchy_version v join staffing.position_hierarchy_outbox e using(instance_id,hierarchy_id,source_id) where source_id=$1", source).Scan(&facts))
				return facts
			}
			before := check()
			must(t, bootstrap.Migrate(ctx, db, migrations.Files))
			must(t, bootstrap.CheckCurrentStorage(ctx, db))
			if before != check() {
				t.Fatal("lost management hierarchy source or event")
			}
			var count int
			must(t, db.QueryRow(ctx, "select count(*) from personnel.person").Scan(&count))
			if count != 0 {
				t.Fatal("invented employee during convergence")
			}
		})
	}
}

// Restore the immediately preceding manual fixture. Compare complete rows,
// including the original profile payloads and the stored submission digests.
func TestPersonnelProfileV31Backup(t *testing.T) {
	checkPersonnelProfileBackup(t, 31)
}

func TestPersonnelProfileV36Backup(t *testing.T) {
	checkPersonnelProfileBackup(t, 36)
}

func TestPersonnelProfileV38Backup(t *testing.T) {
	checkPersonnelProfileBackup(t, 38)
}

func checkPersonnelProfileBackup(t *testing.T, version int) {
	path := os.Getenv("CHNHCM_PERSONNEL_V" + strconv.Itoa(version) + "_DUMP")
	if path == "" {
		t.Skip("requires the matching private acceptance backup")
	}
	f := fresh(t)
	const database = "profile_backup_copy"
	_, err := f.admin.Exec(f.ctx, "create database "+database)
	must(t, err)
	dump, err := os.Open(path)
	must(t, err)
	defer dump.Close()
	cmd := exec.Command("docker", "exec", "-i", f.container.GetContainerID(), "pg_restore", "--exit-on-error", "-U", "postgres", "-d", database)
	cmd.Stdin = dump
	must(t, cmd.Run())
	cfg := f.admin.Config()
	cfg.ConnConfig.Database = database
	db, err := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, err)
	defer db.Close()
	var ledger int
	must(t, db.QueryRow(f.ctx, "select max(version_id) from goose_db_version where is_applied").Scan(&ledger))
	if ledger != version {
		t.Fatalf("input version %d", ledger)
	}
	tables := []string{"person", "number", "counter", "source", "material", "terminal", "outbox", "hold", "disposition", "identity_payload", "evidence_payload", "contact_payload", "document_payload"}
	if version >= 34 {
		tables = append(tables, "profile_item_payload")
	}
	fingerprint := func() map[string]string {
		values := map[string]string{}
		for _, table := range tables {
			var checksum string
			projection := "to_jsonb(v)"
			if table == "profile_item_payload" {
				projection += " - 'account_id'"
			} // 00039 adds a derived reference only for bank rows.
			must(t, db.QueryRow(f.ctx, "select md5(coalesce(string_agg(row_data,'|' order by row_data),'')) from (select ("+projection+")::text as row_data from personnel."+table+" v) original").Scan(&checksum))
			values[table] = checksum
		}
		return values
	}
	before := fingerprint()
	must(t, bootstrap.Migrate(f.ctx, db, migrations.Files))
	must(t, bootstrap.Migrate(f.ctx, db, migrations.Files))
	if !reflect.DeepEqual(before, fingerprint()) {
		t.Fatal("prior profile data changed")
	}
	var instance string
	var count, receipts, items int
	must(t, db.QueryRow(f.ctx, "select instance_id,(select count(*) from personnel.person),(select count(*) from personnel.terminal),(select count(*) from personnel.profile_item_payload) from personnel.person limit 1").Scan(&instance, &count, &receipts, &items))
	if (version == 31 && items != 0) || (version == 36 && items == 0) || count == 0 || receipts == 0 {
		t.Fatal("invalid upgraded fixture")
	}
	u, err := url.Parse(f.runtime.Pool.Config().ConnString())
	must(t, err)
	u.Path = "/" + database
	rt, err := bootstrap.Open(f.ctx, u.String(), instance)
	must(t, err)
	defer rt.Close()
	srv := httptest.NewUnstartedServer(nil)
	srv.Config.Handler = (&bootstrap.API{Runtime: rt, Origin: "http://" + srv.Listener.Addr().String(), InsecureLocal: true}).Handler()
	srv.Start()
	defer srv.Close()
	g := fixture{t: t, ctx: f.ctx, admin: db, instance: instance, server: srv, runtime: rt}
	b := g.login("admin", "admin1234567")
	rows, err := db.Query(f.ctx, "select person_id,number,head from personnel.person where instance_id=$1", instance)
	must(t, err)
	for rows.Next() {
		var id, number, head string
		must(t, rows.Scan(&id, &number, &head))
		detail := b.call("GET", "/api/v1/persons/"+id, nil, "", 200)
		if detail["number"] != number || detail["sourceVersion"] != head || (version == 31 && (detail["family"] != nil || detail["addresses"] != nil)) || (version < 38 && (detail["education"] != nil || detail["credentials"] != nil)) || detail["bankAccounts"] != nil || detail["healthSummary"] != nil {
			t.Fatal("old detail mismatch")
		}
	}
	must(t, rows.Err())
	rows.Close()
	rows, err = db.Query(f.ctx, "select submission_id,version,person_id from personnel.terminal t join access.principal a on (t.instance_id,t.actor)=(a.instance_id,a.principal_id) where a.login='admin' and t.outcome='accepted'")
	must(t, err)
	checked := 0
	for rows.Next() {
		var key, version, id string
		must(t, rows.Scan(&key, &version, &id))
		v := b.call("POST", "/api/v1/persons/submissions/"+key+"/resolve", nil, "", 200)
		if v["id"] != id || v["sourceVersion"] != version {
			t.Fatal("prior receipt mismatch")
		}
		checked++
	}
	must(t, rows.Err())
	rows.Close()
	if checked == 0 {
		t.Fatal("missing accepted receipts")
	}
	must(t, db.QueryRow(f.ctx, "SELECT max(version_id) FROM goose_db_version WHERE is_applied").Scan(&ledger))
	t.Logf("%05d → %05d，%d 张原表一致，%d 人，%d 回执，API 查回 %d 个已接受的原回执，既有地址及家庭行数 %d", version, ledger, len(tables), count, receipts, checked, items)
}

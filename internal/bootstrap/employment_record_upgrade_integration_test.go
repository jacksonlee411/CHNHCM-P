//go:build integration

package bootstrap_test

import (
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestEmploymentRecordUpgradePreservesWithdrawnAndReplacementHires(t *testing.T) {
	ctx := t.Context()
	pool := newStaffingMigrationPool(t)
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00040" })))
	instance, person, employer, department, job, actor := f.NewID(), f.NewID(), f.NewID(), f.NewID(), f.NewID(), f.NewID()
	source, version := f.NewID(), f.NewID()
	tx, err := pool.Begin(ctx)
	must(t, err)
	defer tx.Rollback(ctx)
	exec := func(sql string, args ...any) {
		t.Helper()
		_, err := tx.Exec(ctx, sql, args...)
		must(t, err)
	}
	line, lineRevision := f.NewID(), f.NewID()
	exec(`INSERT INTO staffing.appointment_version(instance_id,source,version,edition,number,roster,withdrawn,members,digest,actor)
		VALUES($1,$2,$3,1,'合成调入名单',true,false,jsonb_build_array(jsonb_build_object('line',$4::uuid,'lineRevision',$5::uuid,'person',$6::uuid)),'合成名单摘要',$7)`, instance, source, version, line, lineRevision, person, actor)
	exec("INSERT INTO staffing.appointment_outbox VALUES($1,$2,$3)", instance, source, version)
	for _, withdrawn := range []bool{true, false} {
		relation, assignment, hire, revision := f.NewID(), f.NewID(), f.NewID(), f.NewID()
		exec("INSERT INTO staffing.employment VALUES($1,$2,$3,$4,'Employee',daterange('2027-04-01',NULL,'[)'))", instance, relation, person, employer)
		exec("INSERT INTO staffing.assignment VALUES($1,$2,$3,$4,$5,NULL,NULL)", instance, assignment, relation, department, job)
		exec(`INSERT INTO staffing.assignment_segment VALUES($1,$2,daterange('2027-04-01',NULL,'[)'),jsonb_build_object('jobRevision',$3::uuid,'seats',0))`, instance, assignment, revision)
		exec(`INSERT INTO staffing.employment_action(instance_id,id,revision,kind,employment_id,assignment_id,business_date,actor,notice,adoption)
			VALUES($1,$2,$3,'Hire',$4,$5,'2027-04-01',$6,'合成入职通知',jsonb_build_object('primaryAssignment',$5::uuid))`, instance, hire, revision, relation, assignment, actor)
		exec("INSERT INTO staffing.hire_terminal VALUES($1,$2,$3,'原摘要','accepted',$4)", instance, actor, f.NewID(), hire)
		exec("INSERT INTO staffing.employment_outbox VALUES($1,$2,$3)", instance, f.NewID(), hire)
		if withdrawn {
			cancel := f.NewID()
			exec("INSERT INTO staffing.hire_decision_use VALUES($1,$2,$3,$4,$5)", instance, source, version, line, hire)
			exec(`INSERT INTO staffing.employment_action(instance_id,id,revision,kind,employment_id,assignment_id,original_id,business_date,actor,notice,adoption)
				VALUES($1,$2,$3,'CancelHire',$4,$5,$6,'2027-04-01',$7,'到岗前撤回',jsonb_build_object('originalRevision',$8::uuid))`, instance, cancel, f.NewID(), relation, assignment, hire, actor, revision)
			exec("INSERT INTO staffing.hire_terminal VALUES($1,$2,$3,'撤回摘要','accepted',$4)", instance, actor, f.NewID(), cancel)
			exec("INSERT INTO staffing.employment_outbox VALUES($1,$2,$3)", instance, f.NewID(), cancel)
		} else {
			exec(`INSERT INTO staffing.employment_primary VALUES($1,$2,'employment',$3,$4,'personnel',daterange('2027-04-01',NULL,'[)')),
				($1,$2,'assignment',$4,$5,'personnel',daterange('2027-04-01',NULL,'[)'))`, instance, hire, person, relation, assignment)
		}
	}
	must(t, tx.Commit(ctx))
	// Compare each original tuple, including the immutable notice, version, time and line.
	tables := []string{"employment", "assignment", "assignment_segment", "employment_action", "hire_terminal", "employment_primary", "employment_outbox", "hire_decision_use", "appointment_version", "appointment_outbox"}
	snapshot := func(table string, upgraded bool) string {
		t.Helper()
		expression := "to_jsonb(t)"
		if table == "assignment" {
			expression += " - 'person_id' - 'record_number' - 'period'"
		}
		from := "staffing." + table + " t"
		if upgraded {
			switch table {
			case "assignment_segment":
				expression += " - 'location_id' - 'location_ref' - 'version' - 'department_id' - 'job_id' - 'grade_ref'"
			case "hire_terminal", "employment_outbox":
				expression += " - 'revision'"
			case "employment_action":
				from += " JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(t.instance_id,t.id,t.revision)"
				expression += " || jsonb_build_object('business_date',r.business_date,'notice',r.notice,'adoption',r.adoption)"
			}
		}
		var data string
		must(t, pool.QueryRow(ctx, "SELECT jsonb_agg(row_data ORDER BY row_data)::text FROM (SELECT "+expression+" AS row_data FROM "+from+") rows").Scan(&data))
		return data
	}
	before := map[string]string{}
	for _, table := range tables {
		before[table] = snapshot(table, false)
	}
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00047" })))
	for _, table := range tables {
		if snapshot(table, true) != before[table] {
			t.Fatalf("00047 前置升级改变了 %s", table)
		}
	}
	publications := func() string {
		var data string
		must(t, pool.QueryRow(ctx, "SELECT jsonb_agg(to_jsonb(p) ORDER BY instance_id,assignment_id,version)::text FROM staffing.assignment_publication p").Scan(&data))
		return data
	}
	originalSources := publications()
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00049" })))
	// A 00049 position already taken over by M04 has no attestation from recruiting.
	previousPosition, source49, enterprise := f.NewID(), f.NewID(), f.NewID()
	old, err := pool.Begin(ctx)
	must(t, err)
	defer old.Rollback(ctx)
	_, err = old.Exec(ctx, `INSERT INTO staffing.position(instance_id,enterprise_id,id,code,source_id,source_no,first_date,head)
      VALUES($1,$2,$3,'OLD49',$4,1,'2027-04-01','{}')`, instance, enterprise, previousPosition, source49)
	must(t, err)
	_, err = old.Exec(ctx, `INSERT INTO staffing.position_source(instance_id,object_id,source_id,source_no,subject_id,submission_id,digest,snapshot,operation,operator)
      VALUES($1,$2,$3,1,$4,$5,decode(repeat('02',32),'hex'),'{}','{"type":"CREATE"}','{}')`, instance, previousPosition, source49, actor, f.NewID())
	must(t, err)
	_, err = old.Exec(ctx, `INSERT INTO staffing.position_intake VALUES($1,$2,$3,daterange('2027-04-01',NULL,'[)'))`, instance, previousPosition, source49)
	must(t, err)
	must(t, old.Commit(ctx))
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00050" })))
	offer50, rev50, sub50 := f.NewID(), f.NewID(), f.NewID()
	t50, e := pool.Begin(ctx)
	must(t, e)
	defer t50.Rollback(ctx)
	_, e = t50.Exec(ctx, `INSERT INTO recruiting.offer(instance_id,id,number,person_id,employer_id,department_id,job_id,arrival,reserve_from,terms) VALUES($1,$2,'旧接受通知',$3,$4,$5,$6,'2027-05-01','2027-04-01','{}')`, instance, offer50, person, employer, department, job)
	must(t, e)
	issue50 := f.NewID()
	_, e = t50.Exec(ctx, `INSERT INTO recruiting.offer_revision(instance_id,offer_id,version,previous,operation,status,business_date,basis,actor) VALUES($1,$2,$3,NULL,'issue','issued','2027-04-01','旧签发人',$5),($1,$2,$4,$3,'accept','accepted','2027-04-03','本人接受',$5)`, instance, offer50, issue50, rev50, actor)
	must(t, e)
	_, e = t50.Exec(ctx, `INSERT INTO recruiting.offer_head VALUES($1,$2,$3)`, instance, offer50, rev50)
	must(t, e)
	_, e = t50.Exec(ctx, `INSERT INTO recruiting.offer_segment VALUES($1,$2,$3,daterange('2027-04-01',NULL,'[)'),NULL,'none','{}')`, instance, offer50, rev50)
	must(t, e)
	_, e = t50.Exec(ctx, `INSERT INTO recruiting.offer_submission(instance_id,actor,submission_id,digest,offer_id,version) VALUES($1,$2,$3,'旧接受回执',$4,$5)`, instance, actor, sub50, offer50, rev50)
	must(t, e)
	must(t, t50.Commit(ctx))
	must(t, bootstrap.Migrate(ctx, pool, hireMigrationSet(t, func(v string) bool { return v <= "00051" })))
	before52 := publications()
	var exchangeRows int
	must(t, pool.QueryRow(ctx, "SELECT count(*) FROM staffing.transfer_exchange_revision").Scan(&exchangeRows))
	for range 2 {
		must(t, bootstrap.Migrate(ctx, pool, migrations.Files))
	}
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
	if publications() != before52 {
		t.Fatal("00052 改变原任期来源")
	}
	var afterExchangeRows int
	must(t, pool.QueryRow(ctx, "SELECT count(*) FROM staffing.transfer_exchange_revision").Scan(&afterExchangeRows))
	if afterExchangeRows != exchangeRows {
		t.Fatal("00052 改变既有对调数")
	}
	var intact bool
	must(t, pool.QueryRow(ctx, `SELECT EXISTS(SELECT FROM recruiting.offer_head h JOIN recruiting.offer_revision r USING(instance_id,offer_id,version)
       JOIN recruiting.offer_segment s USING(instance_id,offer_id,version) JOIN recruiting.offer_submission j USING(instance_id,offer_id,version)
       WHERE h.instance_id=$1 AND h.offer_id=$2 AND r.version=$3 AND r.previous=$4 AND r.status='accepted' AND s.effect='none' AND j.submission_id=$5 AND j.digest='旧接受回执')
       AND NOT EXISTS(SELECT FROM staffing.transfer_exchange WHERE instance_id=$1)`, instance, offer50, rev50, issue50, sub50).Scan(&intact))
	if !intact {
		t.Fatal("00050 原录用、采用和回执在升级后发生变化")
	}
	var sameIntake, hasOfferingSource bool
	must(t, pool.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.position_intake WHERE instance_id=$1 AND position_id=$2 AND source_id=$3),
      EXISTS(SELECT FROM recruiting.position_intake WHERE instance_id=$1 AND position_id=$2)`, instance, previousPosition, source49).Scan(&sameIntake, &hasOfferingSource))
	if !sameIntake || hasOfferingSource {
		t.Fatal("旧岗位接管被更改或被错误判定为招聘已核实")
	}
	if publications() != originalSources {
		t.Fatal("升级改变了旧任期的已发布来源")
	}
	for _, table := range tables {
		if snapshot(table, true) != before[table] {
			t.Fatalf("升级改变了旧数据 %s", table)
		}
	}
	var covered bool
	must(t, pool.QueryRow(ctx, `SELECT bool_and(s.department_id=t.department_id AND s.job_id=t.job_id AND s.grade_ref IS NULL) FROM staffing.assignment_segment s JOIN staffing.assignment t ON (t.instance_id,t.id)=(s.instance_id,s.assignment_id)`).Scan(&covered))
	if !covered {
		t.Fatal("旧采用的部门或职务没有按原任职身份回填")
	}
	must(t, pool.QueryRow(ctx, `SELECT bool_and(a.period=e.period) FROM staffing.assignment a JOIN staffing.employment e ON (a.instance_id,a.employment_id)=(e.instance_id,e.id)`).Scan(&covered))
	if !covered {
		t.Fatal("旧首任期起止日未按原关系回填")
	}

	var records, terms int
	must(t, pool.QueryRow(ctx, `SELECT count(DISTINCT (r.instance_id,r.person_id,r.record_number)),count(*)
		FROM staffing.employment_record r JOIN staffing.assignment a USING(instance_id,person_id,record_number)
		WHERE r.person_id=$1 AND r.record_number=0`, person).Scan(&records, &terms))
	if records != 1 || terms != 2 {
		t.Fatalf("应有同人一份永久 0 号和两个原任期，实际记录 %d 任期 %d", records, terms)
	}
}

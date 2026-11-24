package bootstrap

import (
	accessapp "chnhcm.local/chnhcm/internal/contexts/access/application"
	accessdb "chnhcm.local/chnhcm/internal/contexts/access/infra/postgres"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	orgapp "chnhcm.local/chnhcm/internal/contexts/organization/application"
	countryapp "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	departmentapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	enterpriseapp "chnhcm.local/chnhcm/internal/contexts/organization/application/enterprise"
	geoareaapp "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	legalentityapp "chnhcm.local/chnhcm/internal/contexts/organization/application/legalentity"
	locationapp "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	persondb "chnhcm.local/chnhcm/internal/contexts/personnel/infra/postgres"
	person "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	staffingapp "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	"chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	staffingdb "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staffing "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"io/fs"
	"log/slog"
	"net/url"
	"strings"
	"time"
)

type Runtime struct {
	Hires              hireprep.Hires
	FirstHireRules     hireprep.Rules
	LegalPurposes      org.LegalPurposes
	AssignmentPolicies staffing.AssignmentPolicies
	Personnel          person.Service
	Staffing           staffing.Service
	Pool               *pgxpool.Pool
	Access             access.Service
	Organization       org.Service
	Department         org.DepartmentService
	Country            org.CountryService
	LegalEntity        org.LegalEntityService
	Enterprise         org.EnterpriseService
	Location           org.LocationService
	GeoArea            org.GeoAreaService
	Instance           string
	draftCleanupCancel context.CancelFunc
	draftCleanupDone   chan struct{}
}

// Keep the pool bounded; request deadlines include connection queue time.
const APIConnectionLimit int32 = 12

func Open(ctx context.Context, dsn, instance string) (*Runtime, error) {
	return openRuntime(ctx, dsn, instance, nil)
}

// OpenForDiagnostics installs the offline runner's observer before any pool
// consumer or cleanup goroutine starts. Product entry points use Open.
func OpenForDiagnostics(ctx context.Context, dsn, instance string, tracer pgx.QueryTracer) (*Runtime, error) {
	return openRuntime(ctx, dsn, instance, tracer)
}

func openRuntime(ctx context.Context, dsn, instance string, tracer pgx.QueryTracer) (*Runtime, error) {
	if !foundation.ValidID(instance) {
		return nil, errors.New("CHNHCM_INSTANCE_ID must be a UUID")
	}
	c, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		return nil, errors.New("invalid database connection configuration")
	}
	c.ConnConfig.RuntimeParams["timezone"] = "UTC"
	c.ConnConfig.RuntimeParams["application_name"] = "chnhcm"
	c.MaxConns = APIConnectionLimit
	c.ConnConfig.Tracer = tracer
	p, e := pgxpool.NewWithConfig(ctx, c)
	if e != nil {
		return nil, errors.New("database configuration unavailable")
	}
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, errors.New("database unavailable")
	}
	if e = checkStorage(ctx, p); e != nil {
		p.Close()
		return nil, e
	}
	a := &accessdb.Store{Pool: p, Instance: instance}
	o := &orgdb.Store{Pool: p, Instance: instance, Guard: a.Guard}
	st := &staffingdb.Store{Pool: p, Instance: instance, Guard: a.Guard, Organization: o, PositionDirectory: o}
	st.QuotaPolicies = st.PublishedQuotaPolicies
	o.Staffing = st
	people := &persondb.Store{Pool: p, Authorize: a.Personnel, AdoptArea: func(ctx context.Context, tx pgx.Tx, g access.Guard, id, date string) ([]person.CountryAdoption, string, error) {
		nodes, err := o.AdoptProfileArea(ctx, tx, g, id, date)
		if err != nil {
			return nil, "", err
		}
		var path []person.CountryAdoption
		for _, v := range nodes {
			path = append(path, person.CountryAdoption{ID: v.ID, SourceVersion: v.SourceVersion, RecordID: v.RecordID, Revision: v.Revision, Date: date})
		}
		return path, nodes[len(nodes)-1].Name, nil
	}, AdoptCountry: func(ctx context.Context, tx pgx.Tx, g access.Guard, id, source, date string) (*person.CountryAdoption, string, error) {
		value, err := o.AdoptCountry(ctx, tx, g, id, source, date)
		if err != nil {
			return nil, "", err
		}
		return &person.CountryAdoption{ID: value.ID, SourceVersion: value.SourceVersion, RecordID: value.RecordID, Revision: value.Revision, Date: value.Date}, value.Name, nil
	}}
	st.HirePeople = people
	st.HirePurposes = o
	st.EmploymentAllowed = a.EmploymentAllowed
	o.EmploymentUses = st.EmploymentPurposeUses
	a.AfterLogout = func(ctx context.Context, principal access.Principal) {
		if err := people.DropSession(ctx, instance, principal.SessionID); err != nil {
			slog.Warn("revoked person query cleanup failed")
		}
		if err := o.DropSessionDraft(ctx, principal.ID, principal.SessionID); err != nil {
			slog.Warn("revoked session draft cleanup failed; expiry sweep retains ownership")
		}
		if err := st.DropSession(ctx, principal.SessionID); err != nil {
			slog.Warn("revoked staffing selection cleanup failed")
		}
	}
	runtime := &Runtime{Hires: st, FirstHireRules: st, Personnel: people, LegalPurposes: o, AssignmentPolicies: st, Staffing: staffingapp.Service{Store: st, QuotaStore: st}, Pool: p, Instance: instance, Access: accessapp.Service{Port: a}, Organization: orgapp.Service{Store: o}, Department: departmentapp.Service{Store: o}, Location: locationapp.Service{Store: o}, LegalEntity: legalentityapp.Service{Store: o}, Country: countryapp.Service{Store: o}, Enterprise: enterpriseapp.Service{Store: o}, GeoArea: geoareaapp.Service{Store: o}}
	if e = runtime.Ready(ctx); e != nil {
		p.Close()
		return nil, e
	}
	if e = o.ResetDrafts(ctx); e != nil {
		p.Close()
		return nil, e
	}
	runtime.startDraftCleanup(o, st, people)
	return runtime, nil
}
func (r *Runtime) startDraftCleanup(store *orgdb.Store, staff *staffingdb.Store, people *persondb.Store) {
	ctx, cancel := context.WithCancel(context.Background())
	r.draftCleanupCancel = cancel
	r.draftCleanupDone = make(chan struct{})
	go func() {
		defer close(r.draftCleanupDone)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				bounded, done := context.WithTimeout(ctx, 2*time.Second)
				err := store.SweepDrafts(bounded)
				if err == nil {
					err = people.Sweep(bounded)
				}
				if err == nil {
					err = staff.Sweep(bounded)
				}
				done()
				if err != nil && ctx.Err() == nil {
					slog.Warn("expired selection draft cleanup failed")
				}
			}
		}
	}()
}
func (r *Runtime) Close() {
	if r.draftCleanupCancel != nil {
		r.draftCleanupCancel()
		<-r.draftCleanupDone
	}
	r.Pool.Close()
}
func (r *Runtime) Ready(ctx context.Context) error {
	if err := checkStorage(ctx, r.Pool); err != nil {
		return err
	}
	var ok bool
	e := r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT FROM organization.enterprise_profile WHERE instance_id=$1) AND EXISTS(SELECT FROM access.principal WHERE instance_id=$1 AND state='active') AND EXISTS(SELECT FROM organization.storage_contract WHERE contract='m02-compact-5')`, r.Instance).Scan(&ok)
	if e != nil || !ok {
		return errors.New("runtime initialization incomplete")
	}
	return nil
}
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS) error {
	var exists bool
	if e := pool.QueryRow(ctx, "SELECT to_regclass('public.goose_db_version') IS NOT NULL").Scan(&exists); e != nil {
		return e
	}
	if !exists {
		var initialized bool
		if e := pool.QueryRow(ctx, "SELECT to_regclass('organization.storage_contract') IS NOT NULL OR to_regclass('access.principal') IS NOT NULL").Scan(&initialized); e != nil {
			return e
		}
		if initialized {
			return errors.New("current database has no migration ledger; refusing to reconstruct it")
		}
	}
	if exists {
		var applied int
		if e := pool.QueryRow(ctx, "SELECT count(*) FROM public.goose_db_version WHERE is_applied AND version_id>0").Scan(&applied); e != nil {
			return e
		}
		if applied > 0 {
			if e := checkStorage(ctx, pool, true); e != nil {
				return errors.New("refusing to migrate a different development model; use an isolated fresh database")
			}
		}
	}
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	locker, e := lock.NewPostgresSessionLocker()
	if e != nil {
		return e
	}
	// checkStorage admits only known release ledgers and their retry prefixes.
	// Goose fills the reserved versions without changing prior ledger rows.
	options := []goose.ProviderOption{goose.WithSessionLocker(locker), goose.WithAllowOutofOrder(true)}
	if _, err := fs.Stat(migrations, "00016_location_address.sql"); err == nil {
		options = append(options, goose.WithGoMigrations(locationAddressMigration()), goose.WithExcludeNames([]string{"00016_location_address.sql"}))
	}
	if _, err := fs.Stat(migrations, jobClassificationFile); err == nil {
		options = append(options, goose.WithGoMigrations(jobClassificationMigration(migrations)), goose.WithExcludeNames([]string{jobClassificationFile}))
	}
	for _, step := range []struct {
		name    string
		version int64
	}{{reportingMigrationFile, 26}, {positionQuotaFile, 27}, {quotaConvergenceFile, 28}} {
		if _, err := fs.Stat(migrations, step.name); err == nil {
			options = append(options, goose.WithGoMigrations(quotaLineageMigration(migrations, step.version)), goose.WithExcludeNames([]string{step.name}))
		}
	}
	for _, step := range []struct {
		name    string
		version int64
	}{{"00029_persons.sql", 29}, {"00030_person_primary_document.sql", 30}} {
		if _, err := fs.Stat(migrations, step.name); err == nil {
			options = append(options, goose.WithGoMigrations(personnelLineageMigration(migrations, step.name, step.version)), goose.WithExcludeNames([]string{step.name}))
		}
	}
	provider, e := goose.NewProvider(goose.DialectPostgres, db, migrations, options...)
	if e != nil {
		return e
	}
	_, e = provider.Up(ctx)
	if e != nil {
		return e
	}
	if _, err := fs.Stat(migrations, "00034_person_address_family.sql"); err == nil {
		return persondb.DisposeDue(ctx, pool)
	}
	return nil
}

type FirstHireRuleInput struct {
	Version, From, Until string
}

type InitInput struct {
	EmploymentFixture                                                bool
	FirstHireRule                                                    *FirstHireRuleInput
	HireIdentityFixture                                              bool
	PersonalProfileFixture, SensitiveProfileFixture                  bool
	Instance, EnterpriseCode, EnterpriseName, Start, Login, Password string
	PersonnelProfile                                                 string
	ReadOnly                                                         bool
}

func Initialize(ctx context.Context, pool *pgxpool.Pool, in InitInput) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(484820101)"); e != nil {
		return e
	}
	ent, e := orgdb.Initialize(ctx, tx, in.Instance, in.EnterpriseCode, in.EnterpriseName, in.Start)
	if e != nil {
		return e
	}
	subject, e := accessdb.Initialize(ctx, tx, in.Instance, ent, in.Login, in.Password, in.ReadOnly)
	if e != nil {
		return e
	}
	if cfg := in.FirstHireRule; cfg != nil {
		if e = staffingdb.InitializeFirstHireRule(ctx, tx, in.Instance, subject, cfg.Version, foundation.Period{From: cfg.From, Until: cfg.Until}); e != nil {
			return e
		}
	}
	if e = accessdb.InitializePersonnel(ctx, tx, in.Instance, subject, in.PersonnelProfile); e != nil {
		return e
	}
	if in.EmploymentFixture {
		if e = accessdb.InitializeEmployment(ctx, tx, in.Instance, subject); e != nil {
			return e
		}
	}
	if in.HireIdentityFixture {
		if e = accessdb.InitializeHireIdentity(ctx, tx, in.Instance, subject); e != nil {
			return e
		}
	}
	if in.SensitiveProfileFixture {
		if e = accessdb.InitializeSensitiveProfileFixture(ctx, tx, in.Instance, subject); e != nil {
			return e
		}
	}
	if in.PersonalProfileFixture {
		if e = accessdb.InitializePersonalProfileFixture(ctx, tx, in.Instance, subject); e != nil {
			return e
		}
	}
	if e = orgdb.InitializeEnterpriseMaintenance(ctx, tx, in.Instance, access.OperatorSnapshot{SubjectID: subject, Login: &in.Login, Kind: "local_account", IdentityEvidence: "controlled_initialization"}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

type AccessChange struct{ Instance, Login, Operation, Action string }

type LocalPasswordChange struct{ Instance, Login, PreviousPassword, Password string }

func ChangeLocalPassword(ctx context.Context, p *pgxpool.Pool, in LocalPasswordChange) error {
	return accessdb.ChangeLocalPassword(ctx, p, in.Instance, in.Login, in.PreviousPassword, in.Password)
}

func ChangeAccess(ctx context.Context, p *pgxpool.Pool, in AccessChange) error {
	return accessdb.Change(ctx, p, in.Instance, in.Login, in.Operation, in.Action)
}

// ConfigureLocalRoles is restricted to the upgrader connection; secrets stay in caller memory.
func ConfigureLocalRoles(ctx context.Context, p *pgxpool.Pool, apiPassword string) error {
	for _, row := range []struct{ name, password, group string }{{"chnhcm_api", apiPassword, "chnhcm_app"}} {
		if len(row.password) < 24 {
			return errors.New("runtime database password too short")
		}
		var literal string
		if e := p.QueryRow(ctx, "SELECT quote_literal($1::text)", row.password).Scan(&literal); e != nil {
			return e
		}
		var exists bool
		if e := p.QueryRow(ctx, "SELECT EXISTS(SELECT FROM pg_roles WHERE rolname=$1)", row.name).Scan(&exists); e != nil {
			return e
		}
		verb := "CREATE"
		if exists {
			verb = "ALTER"
		}
		if _, e := p.Exec(ctx, verb+" ROLE "+row.name+" LOGIN NOREPLICATION PASSWORD "+literal); e != nil {
			return errors.New("runtime role configuration failed")
		}
		if _, e := p.Exec(ctx, "GRANT "+row.group+" TO "+row.name); e != nil {
			return e
		}
	}
	return nil
}
func ValidOrigin(origin string, insecure bool) bool {
	u, e := url.Parse(origin)
	if e != nil || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if u.Scheme == "https" && u.Host != "" {
		return true
	}
	return insecure && u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1") && !strings.Contains(origin, "\n")
}

// CheckCurrentStorage applies the runtime's exact layout admission to offline
// verification tools before they inspect or maintain a database. It never upgrades.
func CheckCurrentStorage(ctx context.Context, p *pgxpool.Pool) error {
	return checkStorage(ctx, p)
}

func checkStorage(ctx context.Context, p *pgxpool.Pool, pending ...bool) error {
	var staffingPresent, ok bool
	if err := p.QueryRow(ctx, "SELECT to_regclass('staffing.storage_contract') IS NOT NULL").Scan(&staffingPresent); err != nil {
		return errors.New("staffing storage contract unavailable")
	}
	var staffingContract string
	if staffingPresent {
		if err := p.QueryRow(ctx, "SELECT contract FROM staffing.storage_contract WHERE singleton").Scan(&staffingContract); err != nil {
			return errors.New("staffing storage contract unavailable")
		}
	}
	// A maximum version alone accepts holes in the migration ledger. Goose may
	// retain multiple rows for a version; use its latest state and require the
	// exact current applied set. Only the upgrader admits the same-model prefix
	// for explicit migrations; 00016 converts only location address representation.
	e := p.QueryRow(ctx, `SELECT EXISTS(SELECT FROM organization.storage_contract WHERE singleton AND contract='m02-compact-5') AND
     (SELECT (versions=ARRAY(SELECT generate_series(1,40)::bigint)
          OR ($1 AND (
            (cardinality(versions) BETWEEN 2 AND 39 AND versions=ARRAY(SELECT generate_series(1,cardinality(versions))::bigint))
            -- e shipped 32 (then 33) on 28. A retry may already have installed 29 or 30.
            OR (versions=ARRAY(SELECT generate_series(1,person_prefix)::bigint) || ARRAY(SELECT generate_series(32,least(last_version,33))::bigint)
                AND person_prefix BETWEEN 28 AND 30 AND last_version IN (32,33)
                AND (person_prefix>=29 OR to_regclass('personnel.person') IS NULL)
                AND (person_prefix>=30 OR to_regclass('personnel.current_document_number') IS NULL)
                AND to_regclass('personnel.document_payload') IS NULL)
            -- f applied 36, then 38, on 31+34. Allow only known backfill and retry steps.
            OR (last_version IN (36,38) AND array_remove(versions,38::bigint) IN (ARRAY(SELECT generate_series(1,31)::bigint) || ARRAY[34::bigint,36::bigint], ARRAY(SELECT generate_series(1,32)::bigint) || ARRAY[34::bigint,36::bigint], ARRAY(SELECT generate_series(1,34)::bigint) || ARRAY[36::bigint])
                AND to_regclass('staffing.hire_rule_source') IS NULL
                AND (last_version=36 OR to_regclass('staffing.employment_primary') IS NULL))
            OR (last_version=38 AND versions=ARRAY(SELECT generate_series(1,36)::bigint) || ARRAY[38::bigint]
                AND to_regclass('staffing.employment_primary') IS NULL)
            -- f shipped 34 on 31. A retry may already have installed 32.
            OR (last_version=34 AND versions=ARRAY(SELECT generate_series(1,cardinality(versions)-1)::bigint) || ARRAY[34::bigint]
                AND cardinality(versions) IN (32,33)
                AND (32=ANY(versions) OR to_regclass('organization.legal_purpose_source') IS NULL)
                AND to_regclass('organization.legal_purpose_terminal') IS NULL)
          )))
          AND (NOT (39=ANY(versions)) OR (to_regclass('staffing.hire_single_decision') IS NOT NULL AND to_regclass('staffing.hire_roster_line') IS NOT NULL AND EXISTS(SELECT FROM pg_trigger WHERE tgrelid=to_regclass('staffing.appointment_version') AND tgname='immutable_fact' AND tgenabled='O')))
          AND (NOT (37=ANY(versions)) OR (to_regclass('staffing.employment_primary') IS NOT NULL AND EXISTS(SELECT FROM pg_trigger WHERE tgrelid=to_regclass('staffing.employment_action') AND tgname='immutable_fact' AND tgenabled='O')))
          AND (NOT (35=ANY(versions)) OR (
               EXISTS(SELECT FROM pg_trigger WHERE tgrelid=to_regclass('staffing.hire_rule_source')
                 AND tgname='immutable_source' AND tgenabled='O' AND tgfoid=to_regprocedure('staffing.immutable_fact()'))
               AND EXISTS(SELECT FROM pg_constraint WHERE conrelid=to_regclass('staffing.hire_rule_source')
                 AND conname='hire_rule_period_unique' AND contype='x' AND convalidated)))
          AND (NOT (33=ANY(versions)) OR NOT EXISTS(
                 SELECT FROM (VALUES ('organization.legal_purpose_terminal'),('staffing.assignment_policy_terminal')) r(name)
                 WHERE NOT EXISTS(SELECT FROM pg_trigger WHERE tgrelid=to_regclass(r.name) AND tgname='immutable_terminal' AND tgenabled='O')))
          AND (NOT (32=ANY(versions)) OR (
               NOT EXISTS(SELECT FROM (VALUES ('organization.legal_purpose'),('staffing.assignment_policy')) owner(prefix)
                   CROSS JOIN (VALUES ('_source','immutable_source'),('_member','immutable_member'),('_member','sealed_member'),('_outbox','immutable_event')) guard(suffix,name)
                   WHERE NOT EXISTS(SELECT FROM pg_trigger WHERE tgrelid=to_regclass(owner.prefix || guard.suffix) AND tgname=guard.name AND tgenabled='O'))
               AND EXISTS(SELECT FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND conname='readiness_rule_scope')))
          AND CASE
             WHEN cardinality(versions)>=28 THEN $2='staffing-unified-quota-1'
             WHEN cardinality(versions) IN (26,27) AND $2='staffing-job-classification-1' THEN
                to_regprocedure('access.person_policy_for_use(uuid)') IS NOT NULL AND to_regclass('personnel.person') IS NOT NULL
                AND (cardinality(versions)=26 OR to_regclass('personnel.current_document_number') IS NOT NULL)
             WHEN cardinality(versions)=27 THEN $2 IN ('staffing-quota-1','staffing-allocation-1')
             WHEN cardinality(versions)=26 THEN $2 IN ('staffing-classification-reporting-1','staffing-quota-1')
             WHEN cardinality(versions)=25 THEN $2='staffing-job-classification-1'
             WHEN cardinality(versions)=24 THEN $2 IN ('staffing-job-function-1','staffing-reporting-1')
             WHEN cardinality(versions)>=22 THEN $2='staffing-hierarchy-1'
             WHEN cardinality(versions)=21 THEN $2='staffing-position-1'
             WHEN cardinality(versions)=20 THEN $2='staffing-grade-ladder-2'
             WHEN cardinality(versions)=19 THEN $2='staffing-grade-ladder-1'
             WHEN cardinality(versions)=18 THEN $2='staffing-grade-1'
             WHEN cardinality(versions)=17 THEN $2='staffing-job-foundation-1'
             ELSE $2='' END
         FROM (SELECT array_agg(version_id ORDER BY version_id) versions, max(version_id) last_version,
                    max(version_id) FILTER (WHERE version_id<=31) person_prefix FROM
       (SELECT DISTINCT ON (version_id) version_id,is_applied FROM public.goose_db_version ORDER BY version_id,id DESC) latest
       WHERE is_applied AND version_id>0) ledger)
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=29 AND is_applied) OR (
           to_regprocedure('access.person_policy_for_use(uuid)') IS NOT NULL AND
           NOT EXISTS(SELECT FROM unnest(ARRAY['person','number','source','counter','terminal','identity_payload','evidence_payload','contact_payload','material','outbox','query','hold','disposition']) name WHERE to_regclass('personnel.'||name) IS NULL)))
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=30 AND is_applied) OR to_regclass('personnel.current_document_number') IS NOT NULL OR to_regclass('personnel.document_occupied') IS NOT NULL)
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=40 AND is_applied) OR
          (to_regclass('personnel.bank_account') IS NOT NULL AND to_regclass('personnel.person_bank_number') IS NOT NULL
           AND to_regclass('personnel.person_health_summary') IS NOT NULL AND to_regprocedure('access.person_data_purposes_for_use(uuid)') IS NOT NULL))
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=38 AND is_applied) OR
           (to_regclass('personnel.person_highest_degree') IS NOT NULL AND to_regclass('personnel.person_highest_education') IS NOT NULL AND to_regclass('access.person_qualification_fixture') IS NOT NULL))
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=36 AND is_applied) OR
           (to_regclass('access.hire_identity_fixture') IS NOT NULL AND EXISTS(SELECT FROM pg_constraint WHERE conrelid=to_regclass('access.person_grant') AND conname='person_grant_hire_action' AND convalidated)))
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=34 AND is_applied) OR
           (to_regclass('personnel.person_address_primary') IS NOT NULL AND to_regclass('access.person_address_family_fixture') IS NOT NULL))
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=31 AND is_applied) OR (
           to_regclass('personnel.document_payload') IS NOT NULL AND to_regclass('personnel.document_occupied') IS NOT NULL AND to_regclass('personnel.document_primary') IS NOT NULL
           AND to_regclass('personnel.contact_channel_primary') IS NOT NULL AND to_regclass('access.person_profile_fixture') IS NOT NULL
           AND EXISTS(SELECT FROM pg_attribute WHERE attrelid=to_regclass('access.person_grant') AND attname='item_id' AND NOT attisdropped)
           AND EXISTS(SELECT FROM pg_attribute WHERE attrelid=to_regclass('personnel.identity_payload') AND attname='basic' AND NOT attisdropped)))
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=14 AND is_applied) OR to_regclass('organization.enterprise_profile') IS NOT NULL)
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=15 AND is_applied) OR to_regclass('organization.legal_entity') IS NOT NULL)
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=17 AND is_applied) OR (to_regclass('staffing.job_family') IS NOT NULL AND to_regclass('staffing.job') IS NOT NULL AND to_regclass('staffing.storage_contract') IS NOT NULL))
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=18 AND is_applied) OR to_regclass('staffing.grade') IS NOT NULL)
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=19 AND is_applied) OR (to_regclass('staffing.grade_ladder') IS NOT NULL AND to_regclass('staffing.job_grade_manifest') IS NOT NULL AND to_regclass('staffing.grade_ladder_grade_manifest') IS NOT NULL))
       AND CASE
         WHEN $2='staffing-allocation-1' OR ($2='staffing-quota-1' AND NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=27 AND is_applied)) THEN
           EXISTS (SELECT FROM pg_attribute WHERE attrelid=to_regclass('staffing.position_hierarchy') AND attname='purpose' AND NOT attisdropped)
         WHEN $2='staffing-reporting-1' OR $2<>'staffing-job-classification-1' AND EXISTS(SELECT FROM public.goose_db_version WHERE version_id=26 AND is_applied) THEN
           EXISTS (SELECT FROM pg_constraint c JOIN pg_index i ON i.indexrelid=c.conindid
             WHERE c.conrelid=to_regclass('staffing.position_hierarchy') AND c.contype='u' AND c.convalidated
               AND i.indisvalid AND i.indisready AND pg_get_constraintdef(c.oid)='UNIQUE (instance_id, enterprise_id)')
           AND NOT EXISTS (SELECT FROM pg_attribute WHERE attrelid=to_regclass('staffing.position_hierarchy') AND attname='purpose' AND NOT attisdropped)
         ELSE true END
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=23 AND is_applied) OR to_regclass('organization.geoarea_assigned_code') IS NOT NULL)
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=22 AND is_applied) OR
        (SELECT count(*)=8 FROM (VALUES ('position_hierarchy_node','immutable_node'),('position_hierarchy_node','sealed_version'),('position_hierarchy','immutable_root'),('position_hierarchy_version','immutable_version'),('position_hierarchy_edge','immutable_edge'),('position_hierarchy_edge','sealed_version'),('position_hierarchy_terminal','immutable_terminal'),('position_hierarchy_outbox','immutable_event')) expected(rel,trig)
         JOIN pg_trigger t ON t.tgrelid=to_regclass('staffing.'||expected.rel) AND t.tgname=expected.trig AND t.tgenabled='O' AND t.tgfoid=CASE expected.trig WHEN 'sealed_version' THEN to_regprocedure('staffing.reject_late_hierarchy_member()') ELSE to_regprocedure('staffing.immutable_fact()') END))
       AND (CASE
         WHEN $2 IN ('staffing-unified-quota-1','staffing-allocation-1') OR ($2='staffing-quota-1' AND NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=27 AND is_applied)) THEN
           to_regclass('staffing.quota') IS NOT NULL AND to_regclass('staffing.quota_record') IS NOT NULL
           AND (SELECT count(*)=3 FROM (VALUES ('quota_source','immutable_source'),('quota_outbox','immutable_event'),('quota_terminal','immutable_terminal')) x(rel,trig)
                JOIN pg_trigger t ON t.tgrelid=to_regclass('staffing.'||x.rel) AND t.tgname=x.trig AND t.tgenabled='O' AND t.tgfoid=to_regprocedure('staffing.immutable_fact()'))
           AND (($2='staffing-quota-1') = NOT EXISTS(SELECT FROM pg_attribute WHERE attrelid=to_regclass('staffing.quota') AND attname='position_ref' AND attgenerated='s'))
           AND (($2<>'staffing-unified-quota-1') OR EXISTS(SELECT FROM pg_trigger WHERE tgrelid=to_regclass('staffing.quota_control') AND tgname='immutable_control' AND tgenabled='O' AND tgfoid=to_regprocedure('staffing.immutable_fact()')))
         WHEN $2='staffing-quota-1' THEN
        (SELECT count(*)=6 FROM (VALUES ('quota_source','immutable_source'),('quota_record','immutable_record'),('quota_record','sealed_record'),('quota_control','immutable_control'),('quota_outbox','immutable_event'),('quota_terminal','immutable_terminal')) expected(rel,trig)
         JOIN pg_trigger t ON t.tgrelid=to_regclass('staffing.'||expected.rel) AND t.tgname=expected.trig AND t.tgenabled='O'
         AND t.tgfoid=CASE expected.trig WHEN 'sealed_record' THEN to_regprocedure('staffing.seal_quota_records()') ELSE to_regprocedure('staffing.immutable_fact()') END) ELSE true END)
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=25 AND is_applied) OR
       EXISTS (SELECT FROM pg_trigger WHERE tgrelid=to_regclass('staffing.job_function_legacy') AND tgname='immutable_legacy' AND tgenabled='O' AND tgfoid=to_regprocedure('staffing.immutable_fact()')))
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=24 AND is_applied) OR
         CASE WHEN $2='staffing-reporting-1' THEN NOT EXISTS(SELECT FROM pg_class
           WHERE relnamespace=to_regnamespace('staffing') AND starts_with(relname,'job_function'))
         ELSE (NOT EXISTS(SELECT FROM unnest(ARRAY['job_function','job_function_read','job_function_selection']) AS required(name) WHERE to_regclass('staffing.'||name) IS NULL) AND
          (SELECT count(*)=4 FROM (VALUES ('job_function_source','immutable_source'),('job_function_revision','immutable_revision'),('job_function_terminal','immutable_terminal'),('job_function_outbox','immutable_event')) expected(table_name,trigger_name)
           JOIN pg_trigger t ON t.tgrelid=to_regclass('staffing.'||expected.table_name) AND t.tgname=expected.trigger_name
           WHERE t.tgenabled='O' AND t.tgfoid=to_regprocedure('staffing.immutable_fact()'))) END)
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=21 AND is_applied) OR
         (NOT EXISTS(SELECT FROM unnest(ARRAY['position','position_read','position_selection']) AS required(name) WHERE to_regclass('staffing.'||name) IS NULL) AND
          (SELECT count(*)=4 FROM (VALUES ('position_source','immutable_source'),('position_revision','immutable_revision'),('position_terminal','immutable_terminal'),('position_outbox','immutable_event')) expected(table_name,trigger_name)
           JOIN pg_trigger t ON t.tgrelid=to_regclass('staffing.'||expected.table_name) AND t.tgname=expected.trigger_name
           WHERE t.tgenabled='O' AND t.tgfoid=to_regprocedure('staffing.immutable_fact()'))))
       AND (NOT EXISTS(SELECT FROM public.goose_db_version WHERE version_id=20 AND is_applied) OR
         ((SELECT count(*)=6 FROM (VALUES ('grade_source','immutable_source'),('grade_revision','immutable_revision'),('job_grade_manifest','immutable_manifest'),('grade_ladder_grade_manifest','immutable_manifest'),('job_grade_member','immutable_member'),('grade_ladder_grade_member','immutable_member')) expected(table_name,trigger_name)
 JOIN pg_trigger t ON t.tgrelid=to_regclass('staffing.'||expected.table_name) AND t.tgname=expected.trigger_name
 WHERE t.tgenabled='O' AND t.tgfoid=to_regprocedure('staffing.immutable_fact()')) AND (SELECT count(*)=2 FROM pg_trigger WHERE tgrelid IN (to_regclass('staffing.job_grade_member'),to_regclass('staffing.grade_ladder_grade_member')) AND tgname='member_references' AND tgenabled='O' AND tgfoid=to_regprocedure('staffing.check_grade_member_references()'))))`, len(pending) > 0 && pending[0], staffingContract).Scan(&ok)
	if e != nil || !ok {
		return errors.New("current storage contract does not match this program")
	}
	return nil
}

func DisposeExpiredPersonnel(ctx context.Context, p *pgxpool.Pool) error {
	return persondb.DisposeDue(ctx, p)
}

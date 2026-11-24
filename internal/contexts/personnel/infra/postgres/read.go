package postgres

import (
	d "chnhcm.local/chnhcm/internal/contexts/personnel/domain"
	"chnhcm.local/chnhcm/internal/contexts/personnel/infra/postgres/generated"
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"slices"
	"strings"
	"unicode/utf8"
)

func (s *Store) CreateQuery(ctx context.Context, c f.Credentials, keyword string) (string, error) {
	return run(ctx, s, c, func(r *request) (string, error) {
		if !r.entry().CanRead {
			return "", d.Denied()
		}
		if utf8.RuneCountInString(keyword) > 200 {
			return "", d.Invalid("keyword", "查询条件过长")
		}
		ref := f.NewID()
		_, e := r.tx.Exec(ctx, `insert into personnel.query values($1,$2,$3,$4,$5,clock_timestamp()+interval '30 minutes')`, r.instance, r.a.Principal.ID, r.a.Principal.SessionID, ref, f.NFC(strings.TrimSpace(keyword)))
		return ref, e
	})
}

const directorySQL = `with visible as (
 select b.person_id,n.name,b.number,(x.number is not null and x.number<>b.number and strpos(n.name,$5)=0) historical
 from personnel.person b join personnel.identity_payload n on (b.instance_id,b.person_id,b.head)=(n.instance_id,n.person_id,n.version)
 left join personnel.number x on x.instance_id=b.instance_id and x.person_id=b.person_id and x.number=$6
 where b.instance_id=$1 and b.ordinary_until>clock_timestamp() and ($2 or b.person_id=any($3::uuid[]))
 and not (b.person_id=any(coalesce($4::uuid[],'{}'))) and (strpos(n.name,$5)>0 or x.number is not null)
) `

func (r *request) directory(q p.Query) (string, []any, string, error) {
	scope := r.a.Scope("read", "name", "number")
	if !r.entry().CanRead {
		return "", nil, "", d.Denied()
	}
	keyword := ""
	if q.Ref != "" {
		if !f.ValidID(q.Ref) {
			return "", nil, "", d.Missing()
		}
		e := r.tx.QueryRow(r.ctx, `update personnel.query set expires_at=clock_timestamp()+interval '30 minutes' where instance_id=$1 and subject=$2 and session_id=$3 and ref=$4 and expires_at>clock_timestamp() returning keyword`, r.instance, r.a.Principal.ID, r.a.Principal.SessionID, q.Ref).Scan(&keyword)
		if e != nil {
			return "", nil, "", noRows(e)
		}
	}
	order := map[string]string{"": "number", "number": "number", "name": "name", "-name": "name desc", "-number": "number desc"}[q.Sort]
	if order == "" || !slices.Contains([]int{20, 50, 100}, q.Size) || q.Page < 1 {
		return "", nil, "", d.Invalid("page", "名录查询条件无效")
	}
	number, _ := d.Number(keyword)
	return order + ",person_id", []any{r.instance, scope.All, scope.Include, scope.Exclude, keyword, number}, keyword, nil
}
func (s *Store) List(ctx context.Context, c f.Credentials, q p.Query) (p.Directory, error) {
	return run(ctx, s, c, func(r *request) (p.Directory, error) {
		v := p.Directory{Rows: []p.Summary{}}
		order, args, keyword, e := r.directory(q)
		if e != nil {
			return v, e
		}
		v.Keyword = keyword
		// The left join retains the total even after revocation empties the requested page.
		rows, e := r.tx.Query(ctx, directorySQL+` select t.total,v.person_id,v.name,v.number,v.historical from (select count(*) total from visible)t left join (select * from visible order by `+order+` limit $7 offset $8)v on true order by `+order, append(args, q.Size, (q.Page-1)*q.Size)...)
		if e != nil {
			return v, e
		}
		defer rows.Close()
		for rows.Next() {
			var id, name, number *string
			var historical *bool
			if e = rows.Scan(&v.Total, &id, &name, &number, &historical); e != nil {
				return v, e
			}
			if id != nil {
				v.Rows = append(v.Rows, p.Summary{ID: *id, Name: *name, Number: *number, Historical: *historical})
			}
		}
		return v, rows.Err()
	})
}
func (s *Store) Navigation(ctx context.Context, c f.Credentials, id string, q p.Query) (p.Nav, error) {
	return run(ctx, s, c, func(r *request) (p.Nav, error) {
		var v p.Nav
		order, args, _, e := r.directory(q)
		if e != nil {
			return v, e
		}
		e = r.tx.QueryRow(ctx, directorySQL+`, positions as (select person_id,row_number() over w pos,count(*) over () total,lag(person_id) over w prev,lead(person_id) over w next from visible window w as (order by `+order+`)) select pos,total,coalesce(prev::text,''),coalesce(next::text,'') from positions where person_id=$7`, append(args, id)...).Scan(&v.Position, &v.Total, &v.Prev, &v.Next)
		return v, noRows(e)
	})
}
func (r *request) editable(id string) []string {
	var fields []string
	for _, field := range []string{"name", "evidence", "document", "phone", "email"} {
		if r.allowed("read", id, "number", "name", field) && r.allowed("correct", id, field) && (field != "evidence" || r.allowed("verify", id, "evidence")) {
			fields = append(fields, field)
		}
	}
	for _, key := range append(slices.Concat(d.BasicFields, d.DocumentFields), "longTerm", "documentPrimary", "contactPurpose", "contactPrimary") {
		if r.fieldWrite(id, key) {
			fields = append(fields, key)
		}
	}
	for _, s := range d.ItemSections {
		if !r.itemVisible(s, id, "") {
			continue
		}
		for _, k := range s.Fields {
			if r.itemWrite(s, id, "", k) {
				fields = append(fields, k)
			}
		}
	}
	if r.healthWritable(id) {
		fields = append(fields, "healthSummary")
	}
	return fields
}
func validItemView(view, prefix string) bool {
	id, ok := strings.CutPrefix(view, prefix)
	return ok && f.ValidID(id)
}
func (s *Store) Detail(ctx context.Context, c f.Credentials, id, version, view string) (p.Detail, error) {
	if view != "" && view != "fullDocument" && !(version == "" && (view == "edit" || validItemView(view, "document-") || validItemView(view, "credential-") || validItemView(view, "bank-"))) {
		return p.Detail{}, d.Invalid("view", "无效的查看方式")
	}
	return run(ctx, s, c, func(r *request) (p.Detail, error) {
		return r.detail(id, version, true, view)
	})
}
func (r *request) detail(id, version string, exactSource bool, view string) (p.Detail, error) {
	v, e := r.load(id, version)
	if e != nil {
		return p.Detail{}, e
	}
	history := version != ""
	ordinary := r.visible(id) && r.a.At.Before(v.ordinary)
	if !history && !ordinary {
		return p.Detail{}, d.Missing()
	}
	if history && !r.allowed("history.read", id, "name", "number") {
		ordinary = false
	}
	var held []string
	if history && exactSource {
		rows, e := r.tx.Query(r.ctx, "select category from personnel.hold where instance_id=$1 and person_id=$2 and version=$3 and until_at>clock_timestamp()", r.instance, id, version)
		if e != nil {
			return p.Detail{}, e
		}
		held, e = pgx.CollectRows(rows, pgx.RowTo[string])
		if e != nil {
			return p.Detail{}, e
		}
	}
	can := func(field, category string, ordinaryValid, evidenceValid bool) bool {
		if ordinary && ordinaryValid {
			return r.allowed("read", id, "name", "number", field) && (!history || r.allowed("history.read", id, "name", "number", field))
		}
		return history && exactSource && (evidenceValid && r.a.EvidenceAllowed(id, version, "evidence.read", field) || slices.Contains(held, category) && r.a.EvidenceAllowed(id, version, "hold.read", field))
	}
	identity := can("name", "identity", r.a.At.Before(v.ordinary), r.a.At.Before(v.evidence))
	number := can("number", "identity", r.a.At.Before(v.ordinary), r.a.At.Before(v.evidence))
	evidence := can("evidence", "evidence", r.a.At.Before(v.materialUntil), r.a.At.Before(v.materialEvidenceUntil))
	out := v.Detail
	out.Basic = nil
	out.Nationality = nil
	fullDocument := can("document", "identity", r.a.At.Before(v.ordinary), r.a.At.Before(v.evidence))
	maskedDocument := can("document.masked", "identity", r.a.At.Before(v.ordinary), r.a.At.Before(v.evidence))
	if out.PrimaryDocument != nil {
		switch {
		case !fullDocument && !maskedDocument:
			out.PrimaryDocument = nil
		case fullDocument && (view == "fullDocument" || view == "edit" && r.allowed("correct", id, "document")):
		default:
			out.PrimaryDocument = &p.PrimaryDocument{Kind: out.PrimaryDocument.Kind, Number: "********"}
		}
		out.CanReadDocument = out.PrimaryDocument != nil && fullDocument
	}
	if !identity {
		out.Name = ""
	}
	if !number {
		out.Number = ""
		out.Decision = nil
	}
	if !evidence || (history && !exactSource) {
		out.Evidence = nil
		out.Decision = nil
	}
	if !history {
		if view != "edit" || !r.allowed("correct", id, "evidence") || !r.allowed("verify", id, "evidence") {
			out.Evidence = nil
		}
		out.Decision = nil
		r.projectProfile(v, &out, view)
		r.projectItems(v, &out, view)
		r.projectHealth(v, &out)
		out.Editable = r.editable(id)
		if !r.a.At.Before(v.materialUntil) {
			out.Editable = slices.DeleteFunc(out.Editable, func(v string) bool { return v == "evidence" })
		}
		out.Renumber = r.allowed("renumber", id, "number") && r.allowed("renumber.decide", id, "number")
		out.History = r.allowed("history.read", id, "name", "number")
	}
	for _, x := range v.items {
		valid := r.a.At.Before(x.ordinary)

		if !can(x.Kind, "contacts", valid, r.a.At.Before(x.evidence)) {
			if !can(x.Kind+".masked", "contacts", valid, r.a.At.Before(x.evidence)) {
				continue
			}
			x.Masked = true
			x.Value.Value = d.Mask(x.Kind, x.Value.Value)
		}
		if history || !r.allowed("profile.read", id, "contactPurpose") {
			x.Purpose = ""
		}
		if history || !r.allowed("profile.read", id, "contactPrimary") {
			x.Primary = false
		}
		if !history && view == "edit" {
			x.Locked = r.lockedFields(id, x.ItemID, "contactPurpose", "contactPrimary")
		}
		out.Contacts = append(out.Contacts, x.Contact)
	}
	if history && exactSource {
		rows, err := r.tx.Query(r.ctx, "select distinct category from personnel.disposition where instance_id=$1 and person_id=$2 and version=$3", r.instance, id, version)
		if err != nil {
			return p.Detail{}, err
		}
		categories, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return p.Detail{}, err
		}
		for _, category := range categories {
			fields := map[string][]string{"identity": {"name", "number", "document", "document.masked"}, "evidence": {"evidence"}, "contact.phone": {"phone"}, "contact.email": {"email"}}
			for _, field := range fields[category] {
				if ordinary && r.allowed("read", id, "name", "number", field) && r.allowed("history.read", id, "name", "number", field) || r.a.EvidenceAllowed(id, version, "evidence.read", field) {
					out.Disposed = true
				}
			}

		}
	}
	if !ordinary && !identity && !number && !evidence && out.PrimaryDocument == nil && len(out.Contacts) == 0 && !out.Disposed {
		return p.Detail{}, d.Missing()
	}
	return out, nil
}
func (s *Store) DropSession(ctx context.Context, instance, session string) error {
	return generated.New(s.Pool).DropSessionQueries(ctx, generated.DropSessionQueriesParams{InstanceID: instance, SessionID: session})
}
func (s *Store) Sweep(ctx context.Context) error { return generated.New(s.Pool).ExpireQueries(ctx) }

func (s *Store) ResolveNumber(ctx context.Context, c f.Credentials, input string) (p.Summary, error) {
	return run(ctx, s, c, func(r *request) (p.Summary, error) {
		var v p.Summary
		if requiredPolicy(r) != nil {
			return v, d.Missing()
		}
		number, e := d.Number(input)
		if e != nil {
			return v, e
		}
		scope := r.a.Scope("read", "name", "number")
		e = r.tx.QueryRow(ctx, `select b.person_id,n.name,b.number,b.number<>x.number from personnel.number x
 join personnel.person b using(instance_id,person_id)
 join personnel.identity_payload n on (n.instance_id,n.person_id,n.version)=(b.instance_id,b.person_id,b.head)
 where x.instance_id=$1 and x.number=$2 and b.ordinary_until>clock_timestamp()
 and ($3 or b.person_id=any($4::uuid[])) and not(b.person_id=any(coalesce($5::uuid[],'{}')))`, r.instance, number, scope.All, scope.Include, scope.Exclude).Scan(&v.ID, &v.Name, &v.Number, &v.Historical)
		return v, noRows(e)
	})
}
func (s *Store) GetRecordedNumber(ctx context.Context, c f.Credentials, id, version string) (string, error) {
	if !f.ValidID(version) || !f.ValidID(id) {
		return "", d.Missing()
	}
	source, e := s.Detail(ctx, c, id, version, "")
	if e != nil {
		return "", e
	}
	if source.Number == "" {
		return "", d.Missing()
	}
	return source.Number, nil
}

// RecognizedPeople discloses only directory identity fields for the supplied candidates.
// The consumer intersects these with its own authorized records before sorting or counting.
func (s *Store) RecognizedPeople(ctx context.Context, c f.Credentials, ids []string, ref string) ([]p.Summary, string, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, "", errors.New("person identity lookup requires caller transaction")
	}
	access, _, err := s.Authorize(ctx, tx, c)
	if err != nil {
		return nil, "", err
	}
	r := request{ctx: ctx, tx: tx, a: access, instance: access.Principal.InstanceID}
	if !r.entry().CanRead {
		return nil, "", d.Denied()
	}
	keyword := ""
	if ref != "" {
		if !f.ValidID(ref) {
			return nil, "", d.Missing()
		}
		if err = tx.QueryRow(ctx, `SELECT keyword FROM personnel.query WHERE instance_id=$1 AND subject=$2 AND session_id=$3 AND ref=$4 AND expires_at>clock_timestamp()`, r.instance, access.Principal.ID, access.Principal.SessionID, ref).Scan(&keyword); err != nil {
			return nil, "", noRows(err)
		}
	}
	scope := access.Scope("read", "name", "number")
	number, _ := d.Number(keyword)
	rows, err := tx.Query(ctx, directorySQL+` SELECT person_id,name,number,historical FROM visible WHERE person_id=ANY($7::uuid[])`, r.instance, scope.All, scope.Include, scope.Exclude, keyword, number, ids)
	if err != nil {
		return nil, "", err
	}
	values, err := pgx.CollectRows(rows, pgx.RowToStructByPos[p.Summary])
	return values, keyword, err
}

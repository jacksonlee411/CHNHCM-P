package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
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
	return runAuthorized(ctx, s, c, s.AuthorizeDirectory, func(r *request) (string, error) {
		if requiredPolicy(r) != nil || !r.a.Scope("read", "name", "number").Available() {
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
 where b.instance_id=$1 and b.ordinary_until>$7 and ($2 or b.person_id=any($3::uuid[]))
 and not (b.person_id=any(coalesce($4::uuid[],'{}'))) and (strpos(n.name,$5)>0 or x.number is not null)
) `

func (r *request) directory(q p.Query, renew bool) (string, []any, string, error) {
	scope := r.a.Scope("read", "name", "number")
	if requiredPolicy(r) != nil || !scope.Available() {
		return "", nil, "", d.Denied()
	}
	keyword := ""
	if q.Ref != "" {
		if !f.ValidID(q.Ref) {
			return "", nil, "", d.Missing()
		}
		var e error
		keyword, e = r.queryKeyword(q.Ref, renew)
		if e != nil {
			return "", nil, "", noRows(e)
		}
	}
	order := map[string]string{"": "number", "number": "number", "name": "name", "-name": "name desc", "-number": "number desc"}[q.Sort]
	if order == "" || !slices.Contains([]int{20, 50, 100}, q.Size) || q.Page < 1 {
		return "", nil, "", d.Invalid("page", "名录查询条件无效")
	}
	number, _ := d.Number(keyword)
	return order + ",person_id", []any{r.instance, scope.All, scope.Include, scope.Exclude, keyword, number, r.a.At}, keyword, nil
}
func (r *request) queryKeyword(ref string, renew bool) (string, error) {
	stmt := `SELECT keyword FROM personnel.query WHERE instance_id=$1 AND subject=$2 AND session_id=$3 AND ref=$4 AND expires_at>clock_timestamp()`
	if renew {
		stmt = `UPDATE personnel.query SET expires_at=clock_timestamp()+interval '30 minutes' WHERE instance_id=$1 AND subject=$2 AND session_id=$3 AND ref=$4 AND expires_at>clock_timestamp() RETURNING keyword`
	}
	var keyword string
	e := r.tx.QueryRow(r.ctx, stmt, r.instance, r.a.Principal.ID, r.a.Principal.SessionID, ref).Scan(&keyword)
	return keyword, e
}

func (s *Store) List(ctx context.Context, c f.Credentials, q p.Query) (p.Directory, error) {
	return run(ctx, s, c, func(r *request) (p.Directory, error) {
		return r.list(q, true)
	})
}
func (r *request) list(q p.Query, renew bool) (p.Directory, error) {
	v := p.Directory{Rows: []p.Summary{}}
	order, args, keyword, e := r.directory(q, renew)
	if e != nil {
		return v, e
	}
	return listDirectory(r.ctx, r.tx, q, order, args, keyword)
}
func listDirectory(ctx context.Context, tx pgx.Tx, q p.Query, order string, args []any, keyword string) (p.Directory, error) {
	v := p.Directory{Rows: []p.Summary{}}
	v.Keyword = keyword
	// The left join retains the total even after revocation empties the requested page.
	rows, e := tx.Query(ctx, directorySQL+` select t.total,v.person_id,v.name,v.number,v.historical from (select count(*) total from visible)t left join (select * from visible order by `+order+` limit $8 offset $9)v on true order by `+order, append(args, q.Size, (q.Page-1)*q.Size)...)
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
}

// OpenDirectory resolves the session-owned query, grants, and retention clock
// once for all projections made in this transaction.
func (s *Store) OpenDirectory(ctx context.Context, c f.Credentials, q p.Query) (p.DirectoryLookup, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("directory lookup requires caller transaction")
	}
	access, _, err := s.AuthorizeDirectory(ctx, tx, c)
	if err != nil {
		return nil, err
	}
	r := request{ctx: ctx, tx: tx, a: access, instance: access.Principal.InstanceID}
	order, args, keyword, err := r.directory(q, false)
	if err != nil {
		return nil, err
	}
	return &directoryLookup{ctx: ctx, tx: tx, q: q, store: s, owner: access.Principal, order: order, args: args, keyword: keyword}, nil
}

type directoryLookup struct {
	store          *Store
	owner          access.Principal
	ctx            context.Context
	tx             pgx.Tx
	q              p.Query
	order, keyword string
	args           []any
}

// Renew the exact session-owned reference after the authorized read commits.
// Updating it in a separate statement lets concurrent list and detail reads
// share the reference without serializing their repeatable-read transactions.
func (v *directoryLookup) Renew(ctx context.Context) error {
	tag, err := v.store.Pool.Exec(ctx, `UPDATE personnel.query SET expires_at=clock_timestamp()+interval '30 minutes'
 WHERE instance_id=$1 AND subject=$2 AND session_id=$3 AND ref=$4 AND expires_at>clock_timestamp()`, v.owner.InstanceID, v.owner.ID, v.owner.SessionID, v.q.Ref)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return d.Missing()
	}
	return nil
}

func (s *Store) Navigation(ctx context.Context, c f.Credentials, id string, q p.Query) (p.Nav, error) {
	return run(ctx, s, c, func(r *request) (p.Nav, error) {
		var v p.Nav
		order, args, _, e := r.directory(q, true)
		if e != nil {
			return v, e
		}
		e = r.tx.QueryRow(ctx, directorySQL+`, positions as (select person_id,row_number() over w pos,count(*) over () total,lag(person_id) over w prev,lead(person_id) over w next from visible window w as (order by `+order+`)) select pos,total,coalesce(prev::text,''),coalesce(next::text,'') from positions where person_id=$8`, append(args, id)...).Scan(&v.Position, &v.Total, &v.Prev, &v.Next)
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

// Page expands only caller-authorized rows before counting and paging.
func (v *directoryLookup) Page(keys []p.DirectoryKey, anchor string) (out p.IndexedDirectory, err error) {
	ctx, tx, order, q := v.ctx, v.tx, v.order, v.q
	args := slices.Clone(v.args)
	out.Keyword = v.keyword
	// Without replacement rows or an anchor this is exactly the ordinary
	// personnel page. Avoid expanding and sorting a second copy of the set.
	if len(keys) == 0 && anchor == "" {
		page, err := listDirectory(ctx, tx, q, order, args, v.keyword)
		if err != nil {
			return out, err
		}
		out.Total = page.Total
		for _, summary := range page.Rows {
			out.Rows = append(out.Rows, p.IndexedPerson{Summary: summary, Key: summary.ID})
		}
		return out, nil
	}
	var ids, people []string
	var orders []int
	for _, k := range keys {
		ids = append(ids, k.ID)
		people = append(people, k.Person)
		orders = append(orders, k.Order)
	}
	args = append(args, ids, people, orders)
	base := directorySQL + `, indexed AS (
 SELECT v.*,coalesce(k.id,v.person_id) row_id,coalesce(k.ordinal,0) ordinal FROM visible v
 LEFT JOIN unnest($8::uuid[],$9::uuid[],$10::int[]) k(id,person,ordinal) ON k.person=v.person_id
 ) `
	order += ",ordinal,row_id"
	// Read one neighbour on each side of the requested page. A selected row
	// on this page can return its exact navigation without ranking every row.
	offset := (q.Page - 1) * q.Size
	start := max(0, offset-1)
	rows, err := tx.Query(ctx, base+`SELECT counts.total,v.person_id,v.name,v.number,v.historical,v.row_id
 FROM (SELECT count(*) total FROM indexed)counts LEFT JOIN (SELECT * FROM indexed ORDER BY `+order+` LIMIT $11 OFFSET $12)v ON true ORDER BY `+order, append(args, q.Size+1+offset-start, start)...)
	if err != nil {
		return out, err
	}
	var page []p.IndexedPerson
	for rows.Next() {
		var id, name, number, row *string
		var historical *bool
		if err = rows.Scan(&out.Total, &id, &name, &number, &historical, &row); err != nil {
			rows.Close()
			return out, err
		}
		if id != nil {
			page = append(page, p.IndexedPerson{Summary: p.Summary{ID: *id, Name: *name, Number: *number, Historical: *historical}, Key: *row})
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, err
	}
	first := min(offset-start, len(page))
	end := min(first+q.Size, len(page))
	out.Rows = page[first:end]
	if anchor == "" {
		return out, nil
	}
	for i := first; i < end; i++ {
		if page[i].Key == anchor {
			out.Nav.Position = start + i + 1
			if i > 0 {
				out.Nav.Prev = page[i-1].Key
			}
			if i+1 < len(page) {
				out.Nav.Next = page[i+1].Key
			}
			return out, nil
		}
	}
	err = tx.QueryRow(ctx, base+`, positions AS (
 SELECT row_id,row_number() OVER w pos,lag(row_id) OVER w prev,lead(row_id) OVER w next FROM indexed WINDOW w AS (ORDER BY `+order+`))
 SELECT pos,coalesce(prev::text,''),coalesce(next::text,'') FROM positions WHERE row_id=$11`, append(args, anchor)...).Scan(&out.Nav.Position, &out.Nav.Prev, &out.Nav.Next)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	return out, err
}

// RecognizedPeople discloses only directory identity fields for the supplied candidates.
// The consumer intersects these with its own authorized records before sorting or counting.
func (s *Store) RecognizedPeople(ctx context.Context, c f.Credentials, ids []string, ref string) ([]p.Summary, string, error) {
	v, err := s.OpenDirectory(ctx, c, p.Query{Ref: ref, Page: 1, Size: 20})
	if err != nil {
		return nil, "", err
	}
	return v.Candidates(ids)
}
func (v *directoryLookup) Candidates(ids []string) ([]p.Summary, string, error) {
	if len(ids) == 0 {
		return nil, v.keyword, nil
	}
	rows, err := v.tx.Query(v.ctx, directorySQL+` SELECT person_id,name,number,historical FROM visible WHERE person_id=ANY($8::uuid[])`, append(slices.Clone(v.args), ids)...)
	if err != nil {
		return nil, "", err
	}
	values, err := pgx.CollectRows(rows, pgx.RowToStructByPos[p.Summary])
	return values, v.keyword, err
}

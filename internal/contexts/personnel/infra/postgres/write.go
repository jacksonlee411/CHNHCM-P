package postgres

import (
	d "chnhcm.local/chnhcm/internal/contexts/personnel/domain"
	"chnhcm.local/chnhcm/internal/contexts/personnel/infra/postgres/generated"
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"maps"
	"slices"
	"strings"
)

type terminal struct {
	p.Result
	digest string
	fault  *f.Fault
	status int
}

func (r *request) terminal(id string) (terminal, error) {
	var t terminal
	e := r.tx.QueryRow(r.ctx, `select digest,outcome,coalesce(person_id::text,''),coalesce(version::text,''),fault,coalesce((fault->>'status')::int,0) from personnel.terminal where instance_id=$1 and actor=$2 and submission_id=$3`, r.instance, r.a.Principal.ID, id).Scan(&t.digest, &t.Outcome, &t.ID, &t.Version, &t.fault, &t.status)
	if t.fault != nil {
		t.fault.Status = t.status
	}
	return t, e
}
func (r *request) record(id string, t terminal) error {
	var fault any
	if t.fault != nil {
		fault = map[string]any{"status": t.fault.Status, "code": t.fault.Code, "field": t.fault.Field, "message": t.fault.Message}
	}
	_, e := r.tx.Exec(r.ctx, `insert into personnel.terminal values($1,$2,$3,$4,$5,nullif($6,'')::uuid,nullif($7,'')::uuid,$8)`, r.instance, r.a.Principal.ID, id, t.digest, t.Outcome, t.ID, t.Version, fault)
	return e
}
func (r *request) project(t terminal) (p.Result, error) {
	if t.ID == "" {
		return t.Result, nil
	}
	ok, e := r.readable(t.ID)
	if e != nil {
		return p.Result{}, e
	}
	if t.fault != nil {
		v := p.Result{Outcome: t.Outcome}
		if ok && t.fault.Code == "PERSON_DOCUMENT_USED" && r.allowed("verify", t.ID, "document") {
			v.ExistingPersonID = t.ID
		}
		return v, nil
	}
	if !ok {
		if t.Outcome == "already_registered" {
			return p.Result{Outcome: "rejected"}, nil
		}
		return p.Result{Outcome: t.Outcome}, nil
	}
	if t.Outcome == "already_registered" {
		t.Outcome = "accepted"
		if !r.allowed("verify", t.ID, "evidence") {
			return p.Result{Outcome: "rejected"}, nil
		}
		e = r.tx.QueryRow(r.ctx, `select s.number from personnel.source s join personnel.person p using(instance_id,person_id)
    where s.instance_id=$1 and s.person_id=$2 and s.version=$3 and p.material_until>clock_timestamp()`, r.instance, t.ID, t.Version).Scan(&t.OriginalNumber)
		if errors.Is(e, pgx.ErrNoRows) {
			return p.Result{Outcome: "rejected"}, nil
		}
		if e != nil {
			return p.Result{}, e
		}
	}
	return t.Result, nil
}

func projectedFault(t terminal, result p.Result) error {
	if t.fault.Code != "PERSON_DOCUMENT_USED" {
		return t.fault
	}
	fault := *t.fault
	if result.ExistingPersonID != "" {
		fault.Message = "该主要证件号码已登记，不能重复建档"
	}
	return &p.DocumentConflict{Fault: &fault, ExistingPersonID: result.ExistingPersonID}
}

type completion struct {
	result p.Result
	fault  error
}

func (s *Store) command(ctx context.Context, c f.Credentials, action, id, submission string, input any, execute func(*request) (p.Result, error)) (p.Result, error) {
	if !f.ValidID(submission) {
		return p.Result{}, d.Invalid("submissionId", "提交号无效")
	}
	v, e := run(ctx, s, c, func(r *request) (completion, error) {
		var out completion
		if e := r.lock("submission:" + r.a.Principal.ID + ":" + submission); e != nil {
			return out, e
		}
		digest := hash([]any{action, id, input})
		t, e := r.terminal(submission)
		if e == nil {
			if t.Outcome == "not_applied" {
				return completion{fault: f.Fail(409, "PERSON_NOT_APPLIED", "原提交已确认未执行")}, nil
			}
			if t.digest != digest {
				return completion{fault: d.Conflict()}, nil
			}
			if t.fault != nil {
				if id != "" {
					ok, err := r.readable(id)
					if err != nil {
						return out, err
					}
					if !ok {
						return completion{fault: d.Missing()}, nil
					}
				} else if !r.entry().CanRegister {
					return completion{fault: d.Denied()}, nil
				}
				if correction, ok := input.(p.Correct); ok {
					if err := r.checkItemFault(id, correction, t.fault.Field); err != nil {
						return completion{fault: err}, nil
					}
				}
				out.result, e = r.project(t)
				if e != nil {
					return out, e
				}
				out.fault = projectedFault(t, out.result)
				return out, nil
			}
			out.result, e = r.project(t)
			return out, e
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return out, e
		}
		// Synchronize the number allocator, material bindings and private-payload disposal.
		if e = r.lock("personnel-writes"); e != nil {
			return out, e
		}
		sub, e := r.tx.Begin(ctx)
		if e != nil {
			return out, e
		}
		work := *r
		work.tx = sub
		result, e := execute(&work)
		t = terminal{digest: digest, Result: result}
		if e != nil {
			if err := sub.Rollback(ctx); err != nil {
				return out, err
			}
			if !errors.As(e, &t.fault) {
				return out, e
			}
			t.Outcome = "rejected"
		} else if e = sub.Commit(ctx); e != nil {
			return out, e
		}
		if e = r.record(submission, t); e != nil {
			return out, e
		}
		out.result, e = r.project(t)
		if e == nil && t.fault != nil {
			out.fault = projectedFault(t, out.result)
		}
		return out, e
	})
	if e != nil {
		return p.Result{}, e
	}
	v.result.SubmissionID = submission
	return v.result, v.fault
}
func (s *Store) Resolve(ctx context.Context, c f.Credentials, id string) (p.Result, error) {
	return run(ctx, s, c, func(r *request) (p.Result, error) {
		if !f.ValidID(id) {
			return p.Result{}, d.Missing()
		}
		if e := r.lock("submission:" + r.a.Principal.ID + ":" + id); e != nil {
			return p.Result{}, e
		}
		t, e := r.terminal(id)
		if errors.Is(e, pgx.ErrNoRows) {
			t = terminal{Result: p.Result{Outcome: "not_applied"}}
			e = r.record(id, t)
		}
		if e != nil {
			return p.Result{}, e
		}
		result, e := r.project(t)
		result.SubmissionID = id
		return result, e
	})
}
func (r *request) save(v *snapshot, action, previous string) (p.Result, error) {
	version := f.NewID()
	actor := r.a.Principal
	policy := v.policy
	if r.a.Policy.Version != "" {
		policy = r.a.Policy.Version
	}
	_, e := r.tx.Exec(r.ctx, `insert into personnel.source values($1,$2,$3,nullif($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11,$12)`, r.instance, v.ID, version, previous, v.Number, actor.ID, actor.Login, r.a.At, action, v.Decision, policy, nil)
	if e != nil {
		return p.Result{}, e
	}
	if v.Basic == nil {
		v.Basic = map[string]string{}
	}
	_, e = r.tx.Exec(r.ctx, "insert into personnel.identity_payload(instance_id,person_id,version,name,basic,nationality_adoption) values($1,$2,$3,$4,$5,$6)", r.instance, v.ID, version, v.Name, v.Basic, v.Nationality)
	if e != nil {
		return p.Result{}, e
	}
	for i, x := range v.profileItems {
		if !r.a.At.Before(x.ordinary) {
			continue
		}
		_, e = r.tx.Exec(r.ctx, `insert into personnel.profile_item_payload(instance_id,person_id,version,section,item_id,revision,ordinal,fields,adoptions,ordinary_until,evidence_until,collection_policy) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, r.instance, v.ID, version, x.Section, x.ItemID, x.Revision, i, x.Fields, x.Adoptions, x.ordinary, x.evidence, x.policy)
		if e != nil {
			return p.Result{}, e
		}
	}
	if previous != "" {
		_, e = r.tx.Exec(r.ctx, "update personnel.document_payload set occupied=false where instance_id=$1 and person_id=$2 and version=$3 and occupied", r.instance, v.ID, previous)
		if e != nil {
			return p.Result{}, e
		}
	}
	for i, row := range v.docs {
		attrs := maps.Clone(row.Fields)
		delete(attrs, "documentKind")
		delete(attrs, "documentNumber")
		tag, err := r.tx.Exec(r.ctx, `insert into personnel.document_payload(instance_id,person_id,version,item_id,revision,ordinal,kind,number,attributes,long_term,is_primary,occupied,issuing_country_adoption)
	 values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11,$12) on conflict(instance_id,number) where occupied do nothing`, r.instance, v.ID, version, row.ItemID, row.Revision, i, row.Fields["documentKind"], row.Fields["documentNumber"], attrs, row.LongTerm, row.Primary, row.IssuingCountry)
		if err != nil {
			return p.Result{}, err
		}
		if tag.RowsAffected() == 0 {
			var owner p.Result
			err = r.tx.QueryRow(r.ctx, "select person_id,version from personnel.document_payload where instance_id=$1 and number=$2 and occupied", r.instance, row.Fields["documentNumber"]).Scan(&owner.ID, &owner.Version)
			if err != nil {
				return p.Result{}, err
			}
			return owner, &f.Fault{Status: 409, Code: "PERSON_DOCUMENT_USED", Field: "documentNumber", Message: "无法按此证件办理，请联系人事核对"}
		}
	}
	if v.Evidence != nil && r.a.At.Before(v.materialEvidenceUntil) {
		_, e = r.tx.Exec(r.ctx, "insert into personnel.evidence_payload values($1,$2,$3,$4,$5)", r.instance, v.ID, version, v.Evidence.Kind, v.Evidence.Reference)
		if e != nil {
			return p.Result{}, e
		}
	}
	for i, x := range v.items {
		if !r.a.At.Before(x.evidence) {
			continue
		}
		_, e = r.tx.Exec(r.ctx, "insert into personnel.contact_payload values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)", r.instance, v.ID, version, x.ItemID, x.Revision, i, x.Kind, x.Value.Value, x.ordinary, x.evidence, x.policy, x.Purpose, x.Primary && r.a.At.Before(x.ordinary))
		if e != nil {
			return p.Result{}, e
		}
	}
	if previous == "" {
		_, e = r.tx.Exec(r.ctx, "insert into personnel.person values($1,$2,$3,$4,$5,$6,$7,$8,$9)", r.instance, v.ID, v.Number, version, v.ordinary, v.evidence, r.a.Policy.Version, v.materialUntil, v.materialEvidenceUntil)
	} else {
		_, e = r.tx.Exec(r.ctx, "update personnel.person set number=$3,head=$4 where instance_id=$1 and person_id=$2", r.instance, v.ID, v.Number, version)
	}
	if e != nil {
		return p.Result{}, e
	}
	_, e = r.tx.Exec(r.ctx, "insert into personnel.outbox values($1,$2,$3,$4,$5)", r.instance, f.NewID(), v.ID, version, action)
	return p.Result{Outcome: "accepted", ID: v.ID, Version: version}, e
}
func (r *request) occupy(id, number string) (bool, error) {
	count, err := generated.New(r.tx).OccupyNumber(r.ctx, generated.OccupyNumberParams{InstanceID: r.instance, Number: number, PersonID: id})
	return count == 1, err
}
func (r *request) allocate(id string) (string, error) {
	for {
		var n int
		e := r.tx.QueryRow(r.ctx, `insert into personnel.counter values($1,1) on conflict(instance_id) do update set last_number=personnel.counter.last_number+1 where personnel.counter.last_number<99999999 returning last_number`, r.instance).Scan(&n)
		if errors.Is(e, pgx.ErrNoRows) {
			return "", f.Fail(409, "PERSON_NUMBERS_EXHAUSTED", "自动编号已用尽")
		}
		if e != nil {
			return "", e
		}
		number := fmt.Sprintf("P%08d", n)
		ok, e := r.occupy(id, number)
		if e != nil || ok {
			return number, e
		}
	}
}
func (r *request) item(v p.Value) item {
	return item{policy: r.a.Policy.Version, Contact: p.Contact{Value: v, ItemID: f.NewID(), Revision: f.NewID()}, ordinary: d.Cutoff(r.a.At, r.a.Policy.ContactDays), evidence: d.Cutoff(r.a.At, r.a.Policy.ContactEvidenceDays)}
}
func (r *request) bind(ev p.Evidence, result p.Result, original *string) error {
	_, e := r.tx.Exec(r.ctx, `insert into personnel.material values($1,$2,$3,$4,$5) on conflict(instance_id,material_hash) do nothing`, r.instance, hash(ev), result.ID, original, result.Version)
	return e
}

// The optional document is omitted from the original-content digest when absent,
// preserving the recorded meaning of registration forms from migration 00026.
func registrationDigest(in p.Register) string {
	identity := struct {
		Name     string             `json:"name"`
		Evidence *p.Evidence        `json:"evidence"`
		Document *p.PrimaryDocument `json:"primaryDocument,omitempty"`
	}{in.Name, in.Evidence, in.PrimaryDocument}
	return hash([]any{identity, in.Contacts})
}
func (r *request) previousForm(in p.Register, original string) (p.Result, bool, error) {
	var prior p.Result
	var digest *string
	var usable bool
	e := r.tx.QueryRow(r.ctx, `select m.person_id,m.source_version,m.registration_hash,b.material_until>clock_timestamp() from personnel.material m
 join personnel.person b using(instance_id,person_id) where m.instance_id=$1 and m.material_hash=$2`, r.instance, hash(in.Evidence)).Scan(&prior.ID, &prior.Version, &digest, &usable)
	if errors.Is(e, pgx.ErrNoRows) {
		return p.Result{}, false, nil
	}
	if e != nil {
		return p.Result{}, false, e
	}
	ok, e := r.readable(prior.ID)
	if e != nil {
		return p.Result{}, false, e
	}
	if !ok || !usable || !r.allowed("verify", prior.ID, "evidence") {
		return p.Result{}, false, d.Denied()
	}
	if digest == nil || *digest != original {
		return p.Result{}, false, d.Conflict()
	}
	prior.Outcome = "already_registered"
	return prior, true, nil
}
func (s *Store) Register(ctx context.Context, c f.Credentials, in p.Register) (p.Result, error) {
	name, invalid := d.Name(in.Name)
	if invalid == nil {
		in.Name = name
	}
	fields := []string{"name"}
	if in.Evidence != nil {
		fields = append(fields, "evidence")
		if invalid == nil {
			ref, e := d.Material(in.Evidence.Kind, in.Evidence.Reference)
			invalid = e
			if e == nil {
				in.Evidence = &p.Evidence{Kind: in.Evidence.Kind, Reference: ref}
			}
		}
	}
	if in.PrimaryDocument != nil {
		fields = append(fields, "document")
		if invalid == nil {
			number, e := d.Document(in.PrimaryDocument.Kind, in.PrimaryDocument.Number)
			invalid = e
			if e == nil {
				in.PrimaryDocument = &p.PrimaryDocument{Kind: in.PrimaryDocument.Kind, Number: number}
			}
		}
	}
	for i, v := range in.Contacts {
		fields = append(fields, v.Kind)
		if invalid == nil {
			value, e := d.Contact(v.Kind, v.Value)
			invalid = e
			if e == nil {
				in.Contacts[i].Value = value
			}
		}
	}
	if len(in.Contacts) == 0 {
		in.Contacts = nil
	}
	data := in
	data.SubmissionID = ""
	return s.command(ctx, c, "register", "", in.SubmissionID, data, func(r *request) (p.Result, error) {
		if !r.allowed("register", "", fields...) {
			return p.Result{}, d.Denied()
		}
		if e := requiredPolicy(r); e != nil {
			return p.Result{}, e
		}
		if invalid != nil {
			return p.Result{}, invalid
		}
		var original string
		if in.Evidence != nil {
			original = registrationDigest(in)
			if prior, found, e := r.previousForm(in, original); e != nil || found {
				return prior, e
			}
		}
		v := snapshot{Detail: p.Detail{ID: f.NewID(), Name: in.Name, Evidence: in.Evidence, PrimaryDocument: in.PrimaryDocument}, ordinary: d.Cutoff(r.a.At, r.a.Policy.OrdinaryDays), evidence: d.Cutoff(r.a.At, r.a.Policy.EvidenceDays)}
		if in.PrimaryDocument != nil {
			row := newDocument()
			row.Primary = true
			row.Fields["documentKind"] = in.PrimaryDocument.Kind
			row.Fields["documentNumber"] = in.PrimaryDocument.Number
			v.docs = []p.DocumentRow{row}
		}
		v.materialUntil = d.Cutoff(r.a.At, r.a.Policy.MaterialDays)
		v.materialEvidenceUntil = d.Cutoff(r.a.At, r.a.Policy.MaterialEvidenceDays)
		for _, contact := range in.Contacts {
			v.items = append(v.items, r.item(contact))
		}
		var e error
		v.Number, e = r.allocate(v.ID)
		if e != nil {
			return p.Result{}, e
		}
		result, e := r.save(&v, "register", "")
		if e == nil && in.Evidence != nil {
			e = r.bind(*in.Evidence, result, &original)
		}
		return result, e
	})
}
func (r *request) forUpdate(id, expected string) (snapshot, error) {
	if !r.visible(id) {
		return snapshot{}, d.Missing()
	}
	v, e := r.load(id, "")
	if e != nil {
		return v, e
	}
	if !r.a.At.Before(v.ordinary) {
		return v, d.Missing()
	}
	if v.Version != expected {
		return v, d.Conflict()
	}
	v.Decision = nil
	return v, nil
}
func (s *Store) Correct(ctx context.Context, c f.Credentials, id string, in p.Correct) (p.Result, error) {
	var invalid error
	if in.PrimaryDocument != nil {
		n, e := d.Document(in.PrimaryDocument.Kind, in.PrimaryDocument.Number)
		if e != nil {
			invalid = e
		} else {
			in.PrimaryDocument = &p.PrimaryDocument{Kind: in.PrimaryDocument.Kind, Number: n}
		}
	}
	if in.Name != nil {
		n, e := d.Name(*in.Name)
		if e != nil {
			invalid = e
		} else {
			in.Name = &n
		}
	}
	if in.Evidence != nil {
		n, e := d.Material(in.Evidence.Kind, in.Evidence.Reference)
		if e != nil {
			invalid = e
		} else {
			in.Evidence.Reference = n
		}
	}
	for i, x := range in.Contacts {
		if !x.Remove && (x.ItemID == "" || x.Kind != "" || x.Text != "") {
			v, e := d.Contact(x.Kind, x.Text)
			if e != nil {
				invalid = rowInputError("contacts", i, "value", e)
			} else {
				in.Contacts[i].Text = v
			}
		}
	}
	return s.command(ctx, c, "correct", id, in.SubmissionID, in, func(r *request) (p.Result, error) {
		if !r.visible(id) {
			return p.Result{}, d.Missing()
		}
		if len(r.editable(id)) == 0 {
			return p.Result{}, d.Denied()
		}
		v, e := r.forUpdate(id, in.Expected)
		if e != nil {
			return p.Result{}, e
		}
		fields := []string{}
		if in.PrimaryDocument != nil || in.ClearPrimaryDocument {
			fields = append(fields, "document")
		}
		if in.Name != nil {
			fields = append(fields, "name")
		}
		if in.Evidence != nil {
			if !r.a.At.Before(v.materialUntil) || v.Evidence == nil {
				return p.Result{}, d.Denied()
			}
			fields = append(fields, "evidence")
			if !r.allowed("verify", id, "evidence") {
				return p.Result{}, d.Denied()
			}
		}
		original := hash([]any{v.Name, v.Evidence, v.Basic, v.Nationality, v.docs, v.profileItems, v.itemsForHash()})
		if e := r.applyProfile(&v, in); e != nil {
			return p.Result{}, e
		}
		if e := r.applyItems(&v, d.Addresses, in.Addresses); e != nil {
			return p.Result{}, e
		}
		if e := r.applyItems(&v, d.Family, in.Family); e != nil {
			return p.Result{}, e
		}
		newPrimary := map[string]string{}
		seen := map[string]bool{}
		for _, x := range in.Contacts {
			if !x.Remove && x.Kind != "" {
				fields = append(fields, x.Kind)
			}
			if x.ItemID == "" {
				if x.Remove || x.ExpectedRevision != "" {
					return p.Result{}, d.Invalid("contacts", "请选择原联系方式")
				}
				if e = requiredPolicy(r); e != nil {
					return p.Result{}, e
				}
				contact := r.item(p.Value{Kind: x.Kind, Value: x.Text})
				if err := r.editContactFlags(id, &contact, x, newPrimary); err != nil {
					return p.Result{}, err
				}
				v.items = append(v.items, contact)
				continue
			}
			if seen[x.ItemID] {
				return p.Result{}, d.Invalid("contacts", "同一联系方式重复出现")
			}
			seen[x.ItemID] = true
			i := slices.IndexFunc(v.items, func(v item) bool { return v.ItemID == x.ItemID })
			if i < 0 || !r.a.At.Before(v.items[i].ordinary) || !slices.Contains(r.editable(id), v.items[i].Kind) {
				return p.Result{}, d.Denied()
			}
			fields = append(fields, v.items[i].Kind)
			if v.items[i].Revision != x.ExpectedRevision {
				return p.Result{}, d.Conflict()
			}
			if x.Remove {
				if x.Kind != "" || x.Text != "" || x.Purpose != nil || x.Primary != nil {
					return p.Result{}, d.Invalid("contacts", "移除时只需选择原条目")
				}
				if (v.items[i].Primary && !r.rowWrite(id, v.items[i].ItemID, "contactPrimary")) || (v.items[i].Purpose != "" && !r.rowWrite(id, v.items[i].ItemID, "contactPurpose")) {
					return p.Result{}, d.Denied()
				}
				v.items = slices.Delete(v.items, i, i+1)
			} else {
				if x.Kind != "" && v.items[i].Value != (p.Value{Kind: x.Kind, Value: x.Text}) {
					if v.items[i].Kind != x.Kind && v.items[i].Primary {
						if !r.rowWrite(id, v.items[i].ItemID, "contactPrimary") {
							return p.Result{}, d.Denied()
						}
						v.items[i].Primary = false
					}
					v.items[i].Value = p.Value{Kind: x.Kind, Value: x.Text}
					v.items[i].Revision = f.NewID()
				}
				if e := r.editContactFlags(id, &v.items[i], x, newPrimary); e != nil {
					return p.Result{}, e
				}
			}
		}
		for i := range v.items {
			row := &v.items[i]
			if target := newPrimary[row.Kind]; target != "" && target != row.ItemID && row.Primary && r.a.At.Before(row.ordinary) {
				if !r.rowWrite(id, v.items[i].ItemID, "contactPrimary") || !slices.Contains(r.editable(id), row.Kind) {
					return p.Result{}, d.Denied()
				}
				row.Primary = false
				row.Revision = f.NewID()
			}
		}
		if len(fields) > 0 && (!r.allowed("correct", id, fields...) || !r.allowed("read", id, append([]string{"name", "number"}, fields...)...)) {
			return p.Result{}, d.Denied()
		}
		if invalid != nil {
			return p.Result{}, invalid
		}
		if in.Name != nil {
			v.Name = *in.Name
		}
		if in.Evidence != nil {
			var occupied bool
			e = r.tx.QueryRow(ctx, "select exists(select from personnel.material where instance_id=$1 and material_hash=$2 and person_id<>$3)", r.instance, hash(in.Evidence), id).Scan(&occupied)
			if e != nil {
				return p.Result{}, e
			}
			if occupied {
				return p.Result{}, d.Conflict()
			}
			v.Evidence = in.Evidence
		}
		if original == hash([]any{v.Name, v.Evidence, v.Basic, v.Nationality, v.docs, v.profileItems, v.itemsForHash()}) {
			return p.Result{Outcome: "no_change", ID: id, Version: v.Version}, nil
		}
		result, e := r.save(&v, "correct", v.Version)
		if e != nil && len(in.Documents) > 0 {
			for index, change := range in.Documents {
				if (change.Primary != nil && *change.Primary) || (v.primaryIndex() >= 0 && v.docs[v.primaryIndex()].ItemID == change.ItemID) {
					return p.Result{}, rowInputError("documents", index, "documentNumber", e)
				}
			}
			return p.Result{}, e
		}
		if e == nil && in.Evidence != nil {
			e = r.bind(*in.Evidence, result, nil)
		}
		return result, e
	})
}
func (v snapshot) itemsForHash() []p.Contact {
	items := []p.Contact{}
	for _, x := range v.items {
		items = append(items, x.Contact)
	}
	return items
}
func (s *Store) Renumber(ctx context.Context, c f.Credentials, id string, in p.Renumber) (p.Result, error) {
	n, invalid := d.Number(in.New)
	if invalid == nil {
		in.New = n
	}
	in.Ref = strings.TrimSpace(in.Ref)
	return s.command(ctx, c, "renumber", id, in.SubmissionID, in, func(r *request) (p.Result, error) {
		v, e := r.forUpdate(id, in.Expected)
		if e != nil {
			return p.Result{}, e
		}
		if !r.allowed("renumber", id, "number") || !r.allowed("renumber.decide", id, "number") {
			return p.Result{}, d.Denied()
		}
		if invalid != nil {
			return p.Result{}, invalid
		}
		if e = requiredPolicy(r); e != nil {
			return p.Result{}, e
		}
		if e = d.Decision(r.a.At, in.Reason, in.Ref, in.Until, r.a.Policy.AllowPermanentDecision); e != nil {
			return p.Result{}, e
		}
		if in.Old != v.Number {
			return p.Result{}, d.Conflict()
		}
		if n == v.Number {
			return p.Result{Outcome: "no_change", ID: id, Version: v.Version}, nil
		}
		ok, e := r.occupy(id, n)
		if e != nil {
			return p.Result{}, e
		}
		if !ok {
			return p.Result{}, d.Conflict()
		}
		v.Number = n
		v.Decision = &p.NumberDecision{Old: in.Old, New: n, Reason: in.Reason, Ref: in.Ref, Until: in.Until}
		return r.save(&v, "renumber", v.Version)
	})
}

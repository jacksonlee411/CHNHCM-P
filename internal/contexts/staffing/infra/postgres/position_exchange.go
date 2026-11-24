package postgres

import (
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type exchangeMember struct {
	assignment, expected, position string
	original                       h.ActionRef
	result                         employmentResult
}
type exchangePair struct {
	group   string
	members [2]exchangeMember
}
type exchangeContext struct{}

// During both projections, leave the two members out of each other's old
// occupancy. The final per-position check includes both new published sources.
func exchangePeer(ctx context.Context, owner, assignment, position string) string {
	pair, ok := ctx.Value(exchangeContext{}).(*exchangePair)
	if !ok || owner != "M04" {
		return ""
	}
	for i, m := range pair.members {
		if m.assignment == assignment && (position == m.position || position == pair.members[1-i].position) {
			return pair.members[1-i].assignment
		}
	}
	return ""
}

func (u *unit) exchangeFor(ctx context.Context, action, revision string) (*exchangePair, error) {
	var pair exchangePair
	a, b := &pair.members[0], &pair.members[1]
	err := u.tx.QueryRow(ctx, `SELECT g.id,g.first_position,g.second_position,
        r.first_assignment,r.first_action,r.first_revision,r.second_assignment,r.second_action,r.second_revision
        FROM staffing.transfer_exchange_revision r JOIN staffing.transfer_exchange g ON (g.instance_id,g.id)=(r.instance_id,r.exchange_id)
        WHERE r.instance_id=$1 AND ((r.first_action=$2 AND r.first_revision=$3) OR (r.second_action=$2 AND r.second_revision=$3))`,
		u.store.Instance, action, revision).Scan(&pair.group, &a.position, &b.position, &a.assignment, &a.original.Action, &a.original.Revision, &b.assignment, &b.original.Action, &b.original.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &pair, err
}

func (u *unit) exchangeDisclosure(ctx context.Context, a h.Action, revision string, history bool) (*h.ExchangeLink, error) {
	pair, e := u.exchangeFor(ctx, a.ID, revision)
	if e != nil || pair == nil {
		return nil, e
	}
	link := &h.ExchangeLink{Group: pair.group}
	peer := pair.members[0]
	if peer.original.Action == a.ID {
		peer = pair.members[1]
	}
	other, e := u.action(ctx, peer.original.Action)
	if e != nil {
		return nil, e
	}
	current, e := u.actionDisclosure(ctx, other, history)
	if e != nil || !current.from && !current.to {
		return link, e
	}
	other.Revision = peer.original.Revision
	exact, e := u.actionDisclosure(ctx, other, history)
	if e != nil {
		return nil, e
	}
	if exact.from || exact.to {
		link.Peer = peer.assignment
		link.Original = &peer.original
		if history && current.from && current.to && exact.from && exact.to {
			live, e := u.assignmentIdentity(ctx, peer.assignment)
			if e != nil {
				return nil, e
			}
			latest, e := u.action(ctx, peer.original.Action)
			if e != nil {
				return nil, e
			}
			if latest.Revision == peer.original.Revision && latest.Operation != "Delete" && !live.Cancelled {
				link.PeerSource = live.SourceVersion
				permitted := func(right string) (bool, error) {
					for _, dept := range []string{latest.Department, current.before} {
						ok, e := u.employmentAllowed(ctx, right, latest.Person, latest.Employer, dept, false)
						if e != nil || !ok {
							return false, e
						}
					}
					return true, nil
				}
				link.CanRedate, e = permitted("record.redate")
				if e != nil {
					return nil, e
				}
				link.CanDelete, e = permitted("record.delete")
				if e != nil {
					return nil, e
				}
			}
		}
	}
	return link, nil
}

func (st *Store) ExchangePositions(ctx context.Context, c f.Credentials, id, key string, in h.ExchangeCommand) (h.Receipt, error) {
	return st.employmentCommand(ctx, c, key, []any{"position-exchange", id, in}, func(u *unit) (employmentResult, error) {
		return u.exchangePositions(ctx, id, in)
	})
}

func (u *unit) exchangePositions(ctx context.Context, id string, in h.ExchangeCommand) (out employmentResult, err error) {
	if !f.ValidID(id) || !f.ValidID(in.Peer) || id == in.Peer || !f.ValidID(in.Expected) || !f.ValidID(in.PeerExpected) || !in.Action.Valid() ||
		(in.Operation != "Add" && in.Operation != "Redate" && in.Operation != "Delete") ||
		(in.Operation == "Add") != (in.Action == nil) || (in.Operation == "Delete" && in.BusinessDate != "") || (in.Operation != "Delete" && !f.ValidDate(in.BusinessDate)) ||
		(in.Operation == "Add") != (strings.TrimSpace(in.Notice) != "") || utf8.RuneCountInString(in.Notice) > 1000 {
		return out, f.Fail(400, "EXCHANGE_INPUT", "请核对两份任职、生效日期及对调依据")
	}
	var pair exchangePair
	first := 0
	if in.Action == nil {
		pair.group = f.NewID()
		pair.members[0].assignment = id
		pair.members[1].assignment = in.Peer
	} else {
		saved, e := u.exchangeFor(ctx, in.Action.Action, in.Action.Revision)
		if e != nil {
			return out, e
		}
		if saved == nil {
			return out, assignmentConflict()
		}
		pair = *saved
		if pair.members[1].original.Action == in.Action.Action {
			first = 1
		}
		if pair.members[first].assignment != id || pair.members[1-first].assignment != in.Peer {
			return out, assignmentConflict()
		}
	}
	pair.members[first].expected = in.Expected
	pair.members[1-first].expected = in.PeerExpected
	var people []string
	for _, m := range pair.members {
		a, e := u.assignmentIdentity(ctx, m.assignment)
		if e != nil {
			return out, e
		}
		people = append(people, a.Person)
	}
	slices.Sort(people)
	for _, person := range people {
		if e := lock(ctx, u.tx, false, "staffing.hire.person", u.store.Instance, person); e != nil {
			return out, e
		}
	}
	var allInitial []h.Placement
	var allRecords []savedRecord
	day := in.BusinessDate
	if day == "" {
		orig, e := u.action(ctx, in.Action.Action)
		if e != nil {
			return out, e
		}
		day = orig.BusinessDate
	}
	for i := range pair.members {
		m := &pair.members[i]
		a, e := u.assignmentAt(ctx, m.assignment, day, "")
		if e != nil {
			return out, e
		}
		permit := (recordEdit{operation: in.Operation}).permission("Transfer")
		for _, right := range []string{"read", permit} {
			if e = u.requireEmployment(ctx, right, a.Person, a.Employer, a.Department); e != nil {
				return out, e
			}
		}
		if a.Cancelled || !a.Includes(day) || a.SourceVersion != m.expected {
			return out, assignmentConflict()
		}
		identity, e := u.assignmentIdentity(ctx, m.assignment)
		if e != nil {
			return out, e
		}
		initial, records, e := u.historyRows(ctx, identity)
		if e != nil {
			return out, e
		}
		allInitial = append(allInitial, initial...)
		allRecords = append(allRecords, records...)
		if in.Operation == "Delete" {
			continue
		}
		if m.original.Action != "" {
			for j, r := range records {
				if r.ID == m.original.Action {
					records[j].Removed = true
				}
			}
		}
		project, e := projectRecords(identity, initial, records)
		if e != nil {
			return out, e
		}
		at := slices.IndexFunc(project.Segments, func(p h.Placement) bool { return p.Includes(day) })
		if at < 0 {
			return out, assignmentConflict()
		}
		pos := project.Segments[at].Adoption.Position
		if pos == "" || m.position != "" && m.position != pos {
			return out, f.Fail(409, "EXCHANGE_POSITION", "所选日期的原岗位与对调安排不符")
		}
		m.position = pos
	}
	if people[0] == people[1] {
		return out, f.Fail(409, "EXCHANGE_ASSIGNMENT", "请选择两位员工的有效任职")
	}
	if pair.members[0].position == pair.members[1].position {
		return out, f.Fail(409, "EXCHANGE_POSITION", "请选择分别在两个岗位上的任职")
	}
	if e := u.lockRecordPositions(ctx, allInitial, allRecords); e != nil {
		return out, e
	}
	var before [2]d.QuotaUsage
	for i, m := range pair.members {
		before[i], err = u.PositionUsage(ctx, m.position)
		if err != nil {
			return out, err
		}
	}
	ctx = context.WithValue(ctx, exchangeContext{}, &pair)
	for i := range pair.members {
		m := &pair.members[i]
		edit := recordEdit{RecordMaintenance: h.RecordMaintenance{ExpectedSource: m.expected, ExpectedRevision: m.original.Revision, BusinessDate: in.BusinessDate, Notice: in.Notice}, operation: in.Operation, action: m.original.Action}
		if in.Operation == "Add" {
			edit.kind = "Transfer"
			edit.Position = pair.members[1-i].position
		}
		m.result, err = u.editRecord(ctx, m.assignment, edit)
		if err != nil {
			return out, err
		}
	}
	for i, m := range pair.members {
		after, e := u.PositionUsage(ctx, m.position)
		if e != nil {
			return out, e
		}
		quota, e := u.LoadPositionQuota(ctx, m.position, "")
		if e != nil {
			return out, e
		}
		if e = after.CheckExchange(before[i], quota, pair.members[0].assignment, pair.members[1].assignment); e != nil {
			pos, loadErr := u.Load(ctx, d.Position, m.position, "")
			if loadErr != nil {
				return out, loadErr
			}
			var fault *f.Fault
			if errors.As(e, &fault) && len(fault.Message) >= 10 {
				if at := d.At(pos, fault.Message[:10]); at >= 0 {
					return out, f.Fail(fault.Status, fault.Code, pos.Records[at].Data["name"]+"："+fault.Message)
				}
			}
			return out, e
		}
	}
	a, b := pair.members[0], pair.members[1]
	if in.Operation == "Add" {
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.transfer_exchange(instance_id,id,first_assignment,second_assignment,first_action,second_action,first_position,second_position) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, u.store.Instance, pair.group, a.assignment, b.assignment, a.result.action, b.result.action, a.position, b.position)
		if err != nil {
			return out, err
		}
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.transfer_exchange_revision(instance_id,exchange_id,version,first_assignment,first_action,first_revision,second_assignment,second_action,second_revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, u.store.Instance, pair.group, f.NewID(), a.assignment, a.result.action, a.result.revision, b.assignment, b.result.action, b.result.revision)
	if err != nil {
		return out, err
	}
	other := pair.members[1-first].result
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_outbox(instance_id,event_id,action_id,revision) VALUES($1,$2,$3,$4)`, u.store.Instance, f.NewID(), other.action, other.revision)
	return pair.members[first].result, err
}

func (st *Store) ExchangeOptions(ctx context.Context, c f.Credentials, id string, in h.ExchangeSearch) (out []h.ExchangeOption, err error) {
	out = []h.ExchangeOption{}
	if in.Ref == "" || !f.ValidDate(in.BusinessDate) || in.Page < 1 || in.Page > 1000000 {
		return out, f.Fail(400, "EXCHANGE_INPUT", "请核对姓名、业务日期与页码")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, in.BusinessDate, "")
		if e != nil {
			return e
		}
		if e = u.requireEmployment(ctx, "transfer", a.Person, a.Employer, a.Department); e != nil {
			return e
		}
		own, e := u.assignmentPlacement(ctx, a, in.BusinessDate)
		if e != nil {
			return e
		}
		if own.Adoption.Position == "" {
			return nil
		}
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		ok, e := u.positionReadable(ctx, ent.ID)
		if e != nil || !ok {
			return e
		}
		type candidate struct {
			assignment, source, person, employer, dept, name string
			number                                           int
		}
		rows, e := u.tx.Query(ctx, `SELECT t.id,s.version,t.person_id,rel.employer_id,s.department_id,s.adoption->>'positionName',t.record_number FROM staffing.assignment_head h JOIN staffing.assignment_segment s USING(instance_id,assignment_id,version)
        JOIN staffing.assignment t ON (t.instance_id,t.id)=(s.instance_id,s.assignment_id) JOIN staffing.employment rel ON (rel.instance_id,rel.id)=(t.instance_id,t.employment_id)
        WHERE s.instance_id=$1 AND s.period @> $2::date AND t.person_id<>$3 AND coalesce(s.adoption->>'position','') NOT IN ('',$4)`, st.Instance, in.BusinessDate, a.Person, own.Adoption.Position)
		if e != nil {
			return e
		}
		candidates, e := pgx.CollectRows(rows, func(r pgx.CollectableRow) (v candidate, err error) {
			err = r.Scan(&v.assignment, &v.source, &v.person, &v.employer, &v.dept, &v.name, &v.number)
			return
		})
		if e != nil {
			return e
		}
		var visible []candidate
		var people []string
		for _, v := range candidates {
			allowed := true
			for _, who := range []struct{ person, employer, from, to string }{{v.person, v.employer, v.dept, a.Department}, {a.Person, a.Employer, a.Department, v.dept}} {
				for _, dept := range []string{who.from, who.to} {
					for _, right := range []string{"read", "transfer"} {
						ok, e := u.employmentAllowed(ctx, right, who.person, who.employer, dept, false)
						if e != nil {
							return e
						}
						if !ok {
							allowed = false
						}
					}
				}
			}
			ok, e := u.employmentAllowed(ctx, "directory", v.person, v.employer, v.dept, false)
			if e != nil {
				return e
			}
			if allowed && ok {
				visible = append(visible, v)
				people = append(people, v.person)
			}
		}
		summaries, _, e := st.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), c, people, in.Ref)
		if e != nil {
			return e
		}
		names := map[string]string{}
		for _, p := range summaries {
			names[p.ID] = p.Name
		}
		for _, v := range visible {
			if name, ok := names[v.person]; ok {
				out = append(out, h.ExchangeOption{Assignment: v.assignment, Source: v.source, Name: name, RecordNumber: v.number, Position: v.name, Target: own.Adoption.PositionName})
			}
		}
		slices.SortFunc(out, func(a, b h.ExchangeOption) int {
			return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.RecordNumber, b.RecordNumber), cmp.Compare(a.Assignment, b.Assignment))
		})
		start := min(len(out), (in.Page-1)*20)
		out = out[start:min(start+20, len(out))]
		return nil
	})
	return
}

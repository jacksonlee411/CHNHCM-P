package department

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
)

type digestBytes []byte

func (b *digestBytes) text(s string) {
	*b = binary.AppendUvarint(*b, uint64(len(s)))
	*b = append(*b, s...)
}
func (b *digestBytes) boolean(v bool) {
	if v {
		*b = append(*b, 1)
	} else {
		*b = append(*b, 0)
	}
}

type listPathKey struct{ department, day string }
type listPathPrefix struct {
	text    string
	version [32]byte
	visible bool
}
type listPathValue struct {
	mode, day string
	prefix    *listPathPrefix
	version   [32]byte
}

var restrictedListPathHash = sha256.Sum256([]byte(`"restricted"`))

func (v listPathValue) summary() org.DepartmentPath {
	out := org.DepartmentPath{PathMode: v.mode, PathVersion: hex.EncodeToString(v.version[:])}
	if v.prefix != nil && v.prefix.visible {
		out.Path = borrowedString(&v.prefix.text)
		out.PathBusinessAt = ptr(v.day)
	}
	return out
}
func (p *projector) digestBuffer() digestBytes {
	if p.digestScratch == nil {
		p.digestScratch = make(digestBytes, 0, 1024)
	}
	return p.digestScratch[:0]
}
func (p *projector) digestValue(b digestBytes) [32]byte {
	sum := sha256.Sum256(b)
	p.digestScratch = b[:0]
	return sum
}
func (p *projector) listPathValue(t *d.Timeline, r *d.Record, fields uint16) (listPathValue, error) {
	restricted := listPathValue{mode: "restricted", version: restrictedListPathHash}
	if fields&(readPath|readDate|readStatus) != (readPath | readDate | readStatus) {
		return restricted, nil
	}
	day, mode := p.day, "active"
	if r.State == "inactive" {
		mode, day = "inactive_before_stop", dated.NextDate(r.Date, -1)
	} else if len(t.Records) > 0 && day < t.Records[0].Date {
		mode, day = "future_first", t.Records[0].Date
	}
	prefix, err := p.listPrefix(t.ID, day)
	if err != nil {
		return restricted, err
	}
	if !prefix.visible {
		return restricted, nil
	}
	b := p.digestBuffer()
	b.text("department-list-path/2")
	b = append(b, prefix.version[:]...)
	b.text(mode)
	b.text(day)
	b.text(r.Date)
	b.text(r.End)
	return listPathValue{mode: mode, day: day, prefix: prefix, version: p.digestValue(b)}, nil
}
func (p *projector) listPrefix(id, day string) (*listPathPrefix, error) {
	if err := p.ctx.Err(); err != nil {
		return nil, err
	}
	position := -1
	if id != "" && day == p.day {
		var ok bool
		position, ok = p.s.DepartmentPosition(id)
		if !ok {
			return nil, fmt.Errorf("missing department path node")
		}
		if p.listCurrent == nil {
			p.listCurrent = make([]listPathPrefix, len(p.s.Departments))
			p.listCurrentState = make([]uint8, len(p.s.Departments))
		}
		if p.listCurrentState[position] == 2 {
			return &p.listCurrent[position], nil
		}
		if p.listCurrentState[position] == 1 {
			return nil, fmt.Errorf("cyclic stored department path")
		}
		p.listCurrentState[position] = 1
		defer func() {
			if p.listCurrentState[position] == 1 {
				p.listCurrentState[position] = 0
			}
		}()
	}
	key := listPathKey{id, day}
	var prefix *listPathPrefix
	if position >= 0 {
		prefix = &p.listCurrent[position]
	} else {
		if p.listPrefixes == nil {
			p.listPrefixes = map[listPathKey]*listPathPrefix{}
			p.listVisiting = map[listPathKey]bool{}
		}
		if value, ok := p.listPrefixes[key]; ok {
			return value, nil
		}
		if p.listVisiting[key] {
			return nil, fmt.Errorf("cyclic stored department path")
		}
		p.listVisiting[key] = true
		defer delete(p.listVisiting, key)
		prefix = &listPathPrefix{}
	}
	prefix.visible = true
	var parent *listPathPrefix
	// The enterprise remains the ownership boundary, not a department path label.
	// A named group appears here only when it is a real ancestor department.
	kind, name := "root", ""
	var step d.PathStep
	if id != "" {
		var err error
		snapshot := p.s
		if p.pathSnapshots != nil {
			var ok bool
			snapshot, ok = p.pathSnapshots[day]
			if !ok {
				return nil, fmt.Errorf("missing captured path date")
			}
		}
		step, err = d.ReadPathStep(snapshot, id, day)
		if err != nil {
			return nil, err
		}
		parent, err = p.listPrefix(step.Edge.Parent, day)
		if err != nil {
			return nil, err
		}
		kind, name = "department", step.Record.Data.Name
		prefix.visible = parent.visible
		if p.readSet != nil {
			decision := p.readSet.referenceDecision(id, "department")
			prefix.visible = prefix.visible && decision.Allowed && slices.Contains(decision.ReadFields, "name")
		} else {
			ref := p.ref(id, "department")
			if ref.err != nil {
				return nil, ref.err
			}
			prefix.visible = prefix.visible && ref.readableName
		}
	}
	if prefix.visible {
		b := p.digestBuffer()
		var atom [32]byte
		if id != "" {
			key := nodeKey(step.Timeline, step.Record, step.Edge)
			value, ok := [32]byte{}, false
			if p.listFacts != nil {
				value, ok = p.listFacts.nodes[key]
			}
			if !ok {
				value = nodeDigest(step.Timeline, step.Record, step.Edge)
			}
			atom = value
		} else {
			var root digestBytes
			for _, s := range []string{"department-node-input/1", kind, id, name} {
				root.text(s)
			}
			atom = sha256.Sum256(root)
		}
		b.text("department-path-prefix/2")
		b.boolean(parent != nil)
		if parent != nil {
			b = append(b, parent.version[:]...)
		}
		b = append(b, atom[:]...)

		prefix.version = p.digestValue(b)
		prefix.text = name
		if parent != nil && parent.text != "" {
			prefix.text = parent.text + " / " + name
		}
	}
	if position >= 0 {
		p.listCurrentState[position] = 2
	} else {
		p.listPrefixes[key] = prefix
	}
	return prefix, nil
}

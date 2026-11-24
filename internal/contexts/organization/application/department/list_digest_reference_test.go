package department

// Previous path representation retained only as a path-equivalence witness.
import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
)

// Original string-hash representation, reconstructed from full domain path
// nodes, is an independent byte-for-byte witness for the compact list path.
func referencePath(p *projector, t d.Timeline, r d.Record, fields []string) (org.DepartmentPath, error) {
	restricted := org.DepartmentPath{PathMode: "restricted", PathVersion: fingerprint("restricted")}
	if !slices.Contains(fields, "departmentPath") || !slices.Contains(fields, "status") || !slices.Contains(fields, "effectiveDate") {
		return restricted, nil
	}
	day, mode := p.day, "active"
	if r.State == "inactive" {
		day, mode = dated.NextDate(r.Date, -1), "inactive_before_stop"
	} else if day < t.Records[0].Date {
		day, mode = t.Records[0].Date, "future_first"
	}
	nodes, err := d.PathAt(context.Background(), p.s, t.ID, day, p.enterprise.Name)
	if err != nil {
		return restricted, err
	}
	version, text := "", ""
	for _, n := range nodes {
		if n.Kind != "department" {
			continue
		}
		if n.Kind == "department" {
			ref := p.ref(n.DepartmentID, "department")
			if ref.err != nil {
				return restricted, ref.err
			}
			if !ref.readableName {
				return restricted, nil
			}
		}
		var b digestBytes
		b.text("department-path-prefix/1")
		b.text(version)
		b.text(n.Kind)
		b.text(n.DepartmentID)
		b.text(n.Name)
		if n.Kind == "department" {
			for _, s := range []string{n.Source, n.RecordID, n.Revision, n.Edge.DepartmentID, n.Edge.Source, n.Edge.MemberID, n.Edge.BU} {
				b.text(s)
			}
		}
		version = b.sum()
		if text != "" {
			text += " / "
		}
		text += n.Name
	}
	var b digestBytes
	for _, s := range []string{"department-list-path/1", version, mode, day, r.Date, r.End} {
		b.text(s)
	}
	return org.DepartmentPath{PathMode: mode, PathVersion: b.sum(), Path: ptr(text), PathBusinessAt: ptr(day)}, nil
}

func (b digestBytes) sum() string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

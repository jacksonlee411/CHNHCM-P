package postgres

import (
	"unsafe"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
)

// Count retained typed facts conservatively; shared strings are counted on
// every occurrence. Cache accounting must not serialize a whole second forest.
// Go allocator/map overhead and concurrently retained generations remain part
// of the separately measured process RSS, not a claimed hard memory bound.
func buFactBytes(t d.BusinessUnit) int {
	n := int(unsafe.Sizeof(t)) + 64 + len(t.ID) + len(t.Code) + len(t.Source) + len(t.First) + cap(t.Records)*int(unsafe.Sizeof(d.BURecord{}))
	for _, r := range t.Records {
		n += len(r.ID) + len(r.Revision) + len(r.Date) + len(r.End) + len(r.State) + len(r.Name)
	}
	return n
}

package enterprisecheck

import (
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// Display uses the request's captured business day. Reading or locking the
// enterprise dependency must not sample the clock before dependent locks.
func Display(e org.Enterprise, now time.Time) org.Enterprise {
	if len(e.Names) == 0 {
		return e
	}
	i := dated.IndexAt(e.Names, now.In(foundation.Shanghai).Format("2006-01-02"), func(n *org.EnterpriseName) string { return n.Date })
	if i < 0 {
		i = 0
	}
	e.Name = e.Names[i].Name
	return e
}

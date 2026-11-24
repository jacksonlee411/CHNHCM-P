package migrations

import "embed"

// Files owns executable SQL. bootstrap.Migrate also registers the transactional
// address conversion at 16 and the reporting-branch catalog upgrade at 25.
//
//go:embed *.sql lineage315/*.sql
var Files embed.FS

package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"strings"

	"github.com/pressly/goose/v3"
)

const jobClassificationFile = "00025_job_function_adoption.sql"

// The reporting branch shipped a different migration 24. Bring that exact
// layout through both catalog steps in the transaction that records 25.
func jobClassificationMigration(files fs.FS) *goose.Migration {
	up := func(ctx context.Context, tx *sql.Tx) error {
		var contract string
		if err := tx.QueryRowContext(ctx, "SELECT contract FROM staffing.storage_contract WHERE singleton FOR UPDATE").Scan(&contract); err != nil {
			return err
		}
		runSQL := func(name string) error {
			contents, err := fs.ReadFile(files, name)
			if err != nil {
				return err
			}
			up, _, found := strings.Cut(string(contents), "-- +goose Down")
			if !found {
				return errors.New("missing migration boundary in " + name)
			}
			_, err = tx.ExecContext(ctx, up)
			return err
		}
		switch contract {
		case "staffing-reporting-1":
			if err := runSQL("00024_job_function.sql"); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE staffing.storage_contract SET contract='staffing-job-function-1' WHERE singleton"); err != nil {
				return err
			}
		case "staffing-job-function-1": // Mainline already installed the catalog.
		default:
			return errors.New("unrecognized staffing migration 24")
		}
		return runSQL(jobClassificationFile)
	}
	return goose.NewGoMigration(25, &goose.GoFunc{RunTx: up}, &goose.GoFunc{RunTx: func(context.Context, *sql.Tx) error {
		return errors.New("job classification evidence cannot be downgraded")
	}})
}

// Read-only verification of complete decoded immutable organization sources.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("CHNHCM_ADMIN_DATABASE_URL"))
	if err != nil {
		panic("database configuration invalid")
	}
	defer pool.Close()
	encoder := json.NewEncoder(os.Stdout)
	err = bootstrap.VisitHistory(ctx, pool, os.Getenv("CHNHCM_INSTANCE_ID"), func(b *bootstrap.BUHistoryFact, d *bootstrap.DepartmentHistoryFact) error {
		var fact any
		var key string
		if b != nil {
			fact = b
			key = "BU/" + b.Object + "/" + b.Operation.Source.ID
		} else {
			fact = d
			key = "department/" + d.Object + "/" + d.Operation.Source.ID
		}
		raw, err := json.Marshal(fact)
		if err != nil {
			return err
		}
		return encoder.Encode([]string{key, fmt.Sprintf("%x", sha256.Sum256(raw))})
	})
	if err != nil {
		panic("complete history decoding failed: " + err.Error())
	}
}

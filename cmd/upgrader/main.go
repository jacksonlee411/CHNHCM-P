package main

import (
	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"time"
)

func main() {
	mode := flag.String("mode", "migrate", "migrate, initialize, runtime-roles, access-admin, access-change, local-password, country-catalog, geoarea-catalog, personnel-retention, probation-reconcile, t04-reconcile; inputs are JSON on stdin")
	apply := flag.Bool("apply", false, "apply the selected reconciliation; defaults to preview")
	optionalCatalog := flag.Bool("skip-unauthorized-catalog", false, "during local startup, skip a catalog when the account lacks current access")
	flag.Parse()
	reconcile := *mode == "probation-reconcile" || *mode == "t04-reconcile"
	if *apply && !reconcile {
		fail("-apply requires a reconciliation mode")
	}
	budget := time.Minute
	if *mode == "country-catalog" || *mode == "geoarea-catalog" || reconcile {
		budget = 20 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	if *mode == "country-catalog" || *mode == "geoarea-catalog" || reconcile {
		var input access.LoginInput
		decoder := json.NewDecoder(os.Stdin)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			fail("invalid operator login input")
		}
		runtime, err := bootstrap.Open(ctx, os.Getenv("CHNHCM_DATABASE_URL"), os.Getenv("CHNHCM_INSTANCE_ID"))
		if err != nil {
			fail(err.Error())
		}
		defer runtime.Close()
		if reconcile {
			run := runtime.ReconcileProbation
			if *mode == "t04-reconcile" {
				run = runtime.ReconcileT04
			}
			report, err := run(ctx, input, *apply)
			if err != nil {
				fail(err.Error())
			}
			if err = json.NewEncoder(os.Stdout).Encode(report); err != nil {
				fail(err.Error())
			}
			if report.Unresolved+report.Failed > 0 {
				fail(*mode + " requires follow-up")
			}
			return
		}
		if *mode == "geoarea-catalog" {
			err = runtime.InitializeGeoAreaCatalog(ctx, input)
		} else {
			err = runtime.InitializeCountryCatalog(ctx, input)
		}
		if err != nil {
			var fault *foundation.Fault
			if *optionalCatalog && errors.As(err, &fault) && (fault.Code == "ACCESS_DENIED" || fault.Code == "AUTH_REQUIRED") {
				fmt.Println("upgrader: " + *mode + " skipped; account lacks current access")
				return
			}
			fail(*mode + " initialization failed: " + err.Error())
		}
		fmt.Println("upgrader: " + *mode + " completed")
		return
	}
	dsn := os.Getenv("CHNHCM_ADMIN_DATABASE_URL")
	if dsn == "" {
		fail("CHNHCM_ADMIN_DATABASE_URL is required")
	}
	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fail("invalid admin connection configuration")
	}
	defer p.Close()
	switch *mode {
	case "personnel-retention":
		err = bootstrap.DisposeExpiredPersonnel(ctx, p)
	case "migrate":
		err = bootstrap.Migrate(ctx, p, migrations.Files)
	case "initialize":
		var in bootstrap.InitInput
		d := json.NewDecoder(os.Stdin)
		d.DisallowUnknownFields()
		if err = d.Decode(&in); err == nil {
			if (in.DevelopmentFixture || in.NewAccountFixture != nil) && (os.Getenv("CHNHCM_INSECURE_LOCAL") != "1" || os.Getenv("CHNHCM_TEMPORARY") != "true") {
				fail("development fixtures require an explicit temporary local instance")
			}
			err = bootstrap.Initialize(ctx, p, in)
		}
	case "access-admin":
		var in access.RecoveryCommand
		d := json.NewDecoder(os.Stdin)
		d.DisallowUnknownFields()
		if err = d.Decode(&in); err == nil {
			var result access.ManagementResult
			result, err = bootstrap.RecoverManagement(ctx, p, os.Getenv("CHNHCM_INSTANCE_ID"), in)
			if err == nil {
				err = json.NewEncoder(os.Stdout).Encode(result)
				if err == nil {
					return
				}
			}
		}
	case "access-change":
		var in bootstrap.AccessChange
		d := json.NewDecoder(os.Stdin)
		d.DisallowUnknownFields()
		if err = d.Decode(&in); err == nil {
			err = bootstrap.ChangeAccess(ctx, p, in)
		}
	case "local-password":
		if os.Getenv("CHNHCM_INSECURE_LOCAL") != "1" {
			fail("local-password requires explicit local development mode")
		}
		var in bootstrap.LocalPasswordChange
		d := json.NewDecoder(os.Stdin)
		d.DisallowUnknownFields()
		if err = d.Decode(&in); err == nil {
			err = bootstrap.ChangeLocalPassword(ctx, p, in)
		}
	case "runtime-roles":
		var in struct{ API string }
		if err = json.NewDecoder(os.Stdin).Decode(&in); err == nil {
			err = bootstrap.ConfigureLocalRoles(ctx, p, in.API)
		}
	default:
		fail("unknown upgrader mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "upgrader failed:", err)
		os.Exit(1)
	}
	fmt.Println("upgrader:", *mode, "completed")
}
func fail(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }

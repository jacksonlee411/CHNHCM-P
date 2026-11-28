package postgres

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Only the deployment initializer creates this declaration. An account opened
// in Security Management has a different digest, even if its old login was admin.
func initializedAdminID(ctx context.Context, tx pgx.Tx, instance, enterprise string) (string, error) {
	operator := initializationDigest("admin", enterprise, false)
	reader := initializationDigest("admin", enterprise, true)
	var id string
	err := tx.QueryRow(ctx, `SELECT principal_id::text FROM access.principal
 WHERE instance_id=$1 AND normalized_login='admin' AND initialization_digest IN ($2,$3)`,
		instance, operator[:], reader[:]).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// InitializeManagement runs only under the upgrader's database identity.
// The first operator supplies receipt storage until a customer appoints their
// security administrator. An existing appointment and its key are preserved.
func InitializeManagement(ctx context.Context, tx pgx.Tx, instance, enterprise string) error {
	if err := lockManager(ctx, tx, instance, true); err != nil {
		return err
	}
	var configured bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT FROM access.management_root WHERE instance_id=$1)", instance).Scan(&configured); err != nil || configured {
		return err
	}
	id, err := initializedAdminID(ctx, tx, instance, enterprise)
	if err != nil || id == "" {
		return err
	}
	return insertManagementRoot(ctx, tx, instance, id, 0)
}

func insertManagementRoot(ctx context.Context, tx pgx.Tx, instance, subject string, proxyDays int) error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, "INSERT INTO access.management_root(instance_id,principal_id,max_proxy_days,receipt_key) VALUES($1,$2,$3,$4)", instance, subject, proxyDays, key)
	return err
}

const explicitAppointment = `EXISTS(SELECT FROM access.management_receipt
 WHERE instance_id=$1 AND actor_id IS NULL AND operation='admin.bootstrap')`

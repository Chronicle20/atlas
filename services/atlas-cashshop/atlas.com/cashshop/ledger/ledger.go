// Package ledger claims transaction-scoped idempotency for cash shop
// commands. It is a thin wrapper over the shared idempotency table
// (libs/atlas-database's IdempotencyEntity / idempotency_keys) rather than a
// second, purpose-built table: the key is globally unique and the row's only
// job is to be that claim.
package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ErrAlreadyProcessed means this (tenant, transaction) pair has already been
// committed -- a Kafka redelivery, not a new click. Callers treat it as
// success-without-effect, not as a failure to report to the client.
var ErrAlreadyProcessed = errors.New("command already processed for this transaction")

// Claim writes the uniqueness claim for one cash shop command. It MUST be the
// first statement inside the command's transaction, so a redelivery aborts
// before any state changes.
//
// tx is the caller's transaction handle -- Claim joins it, it does not open
// one.
//
// The claim keys on the bare transaction id, not (transaction id, command
// type): a replay of the same transaction under a different commandType is
// still rejected, because it is still the same at-least-once redelivery of
// the same click. Do not derive the key with database.Key() -- that hashes
// operation and payload in, which would let one transaction id through twice
// under two different command types.
//
// characterId is not persisted -- IdempotencyEntity has no column for it --
// but is kept in the signature for callers' own audit logging around the
// call site.
func Claim(ctx context.Context, tx *gorm.DB, transactionId uuid.UUID, commandType string, characterId uint32) error {
	if transactionId == uuid.Nil {
		return fmt.Errorf("ledger: cannot claim the zero transaction id for command [%s], character [%d]: RequestPurchaseCommandBody documents uuid.Nil as \"no correlation\", so it must never become a shared uniqueness claim", commandType, characterId)
	}

	t, err := tenant.FromContext(ctx)()
	if err != nil {
		return fmt.Errorf("ledger: claiming a command requires a tenant in context: %w", err)
	}

	claim := database.IdempotencyEntity{
		TenantId:  t.Id(),
		Key:       transactionId.String(),
		Operation: commandType,
		CreatedAt: time.Now(),
	}
	res := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&claim)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAlreadyProcessed
	}
	return nil
}

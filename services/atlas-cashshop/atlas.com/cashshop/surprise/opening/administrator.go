package opening

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrAlreadyOpened means this (tenant, transaction) pair has already been
// committed — a Kafka redelivery, not a new click. Callers treat it as
// success-without-effect rather than as a failure to report to the client.
var ErrAlreadyOpened = errors.New("surprise box already opened for this transaction")

// Insert writes the ledger row. It MUST be the first statement in the open
// transaction so a duplicate aborts before any state changes.
//
// Duplicate detection relies on the (tenant_id, transaction_id) primary key
// constraint, not a read-then-write check — a SELECT-then-INSERT has a race
// window where two concurrent redeliveries both observe "not present" and
// both insert. The constraint violation is classified by isDuplicateKeyError
// (see duplicate.go), which recognizes it under both drivers this service
// runs against: Postgres SQLSTATE 23505 in production, sqlite extended codes
// 1555/2067 in tests.
func Insert(db *gorm.DB, tenantId uuid.UUID, transactionId uuid.UUID, accountId uint32, assetId uint32) error {
	e := entity{
		TenantId:      tenantId,
		TransactionId: transactionId,
		AccountId:     accountId,
		AssetId:       assetId,
		CreatedAt:     time.Now(),
	}
	err := db.Create(&e).Error
	if err == nil {
		return nil
	}
	if isDuplicateKeyError(err) {
		return ErrAlreadyOpened
	}
	return err
}

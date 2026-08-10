package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ErrDuplicate reports that the work guarded by Once has already been applied
// under the same key. It is an expected outcome, not a failure: Kafka delivery
// in this system is at-least-once (see libs/atlas-kafka/consumer/manager.go and
// libs/atlas-outbox/drainer.go), so any handler that creates durable state will
// eventually see the same command twice.
var ErrDuplicate = errors.New("database: work already applied for this idempotency key")

// IdempotencyEntity records one applied unit of work. The composite primary key
// (tenant_id, key) is the uniqueness constraint that makes a second delivery a
// no-op rather than a second row.
type IdempotencyEntity struct {
	TenantId  uuid.UUID `gorm:"primaryKey;not null"`
	Key       string    `gorm:"primaryKey;not null"`
	Operation string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null;index"`
}

func (e IdempotencyEntity) TableName() string {
	return "idempotency_keys"
}

// IdempotencyMigration creates the idempotency_keys table. Services that guard
// handlers with Once must include it in their migration list.
func IdempotencyMigration(db *gorm.DB) error {
	return db.AutoMigrate(&IdempotencyEntity{})
}

// Key derives a stable idempotency key from the identity of a command: the
// saga/transaction it belongs to, the operation it performs, and its payload.
// A redelivered message is byte-identical and therefore yields the same key; a
// genuinely different command in the same transaction does not.
func Key(transactionId uuid.UUID, operation string, payload any) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("database: unable to encode idempotency payload: %w", err)
	}
	h := sha256.New()
	h.Write(transactionId[:])
	h.Write([]byte{0})
	h.Write([]byte(operation))
	h.Write([]byte{0})
	h.Write(encoded)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Once runs fn exactly once per (tenant, key), returning ErrDuplicate when the
// key has already been applied. The claim row and fn's writes share a single
// transaction: if fn fails the claim rolls back with it, so a failed apply stays
// retryable rather than being permanently swallowed.
//
// fn receives the transaction handle and must use it for its writes. Processors
// built on ExecuteTransaction join it rather than opening a nested one.
func Once(ctx context.Context, db *gorm.DB, key string, operation string, fn func(tx *gorm.DB) error) error {
	t, err := tenant.FromContext(ctx)()
	if err != nil {
		return fmt.Errorf("database: idempotent work requires a tenant in context: %w", err)
	}

	return ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
		claim := IdempotencyEntity{
			TenantId:  t.Id(),
			Key:       key,
			Operation: operation,
			CreatedAt: time.Now(),
		}
		res := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&claim)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrDuplicate
		}
		return fn(tx)
	})
}

// SweepIdempotency deletes claim rows older than retention. Keys only need to
// outlive the window in which a broker or outbox can redeliver; keeping them
// forever would grow the table without bound.
func SweepIdempotency(ctx context.Context, db *gorm.DB, retention time.Duration) error {
	cutoff := time.Now().Add(-retention)
	return db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&IdempotencyEntity{}).Error
}

// ApplyOnce is the command-handler front door to Once: it derives the key from
// the command's identity, runs the work once, and turns a repeat delivery into
// a logged no-op rather than an error the caller has to classify.
//
// If the key cannot be derived the work is applied WITHOUT the guard, loudly.
// Degrading to today's at-least-once behavior risks a duplicate; skipping would
// risk losing the item outright, which is strictly worse.
func ApplyOnce(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, transactionId uuid.UUID, operation string, payload any, fn func(tx *gorm.DB) error) error {
	key, err := Key(transactionId, operation, payload)
	if err != nil {
		l.WithError(err).Errorf("Unable to derive idempotency key for [%s] in transaction [%s]; applying unguarded.", operation, transactionId)
		return fn(db.WithContext(ctx))
	}

	err = Once(ctx, db, key, operation, fn)
	if errors.Is(err, ErrDuplicate) {
		l.Infof("Skipping duplicate [%s] delivery for transaction [%s]; already applied.", operation, transactionId)
		return nil
	}
	return err
}

// Idempotency sweeper defaults. Retention comfortably exceeds any realistic
// redelivery window (broker retention, outbox backlog, a long rebalance) while
// keeping the table small.
const (
	DefaultIdempotencyRetention = 7 * 24 * time.Hour
	DefaultIdempotencySweep     = time.Hour
)

// StartIdempotencySweeper runs SweepIdempotency on an interval until ctx is
// canceled. The delete is unconditional and idempotent, so every replica may
// run it — no leader election required. The sweep is cross-tenant, so it runs
// with tenant filtering disabled.
func StartIdempotencySweeper(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, retention time.Duration, interval time.Duration) {
	routine.Go(l, ctx, func(rctx context.Context) {
		sweepCtx := WithoutTenantFilter(rctx)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-rctx.Done():
				return
			case <-t.C:
				if err := SweepIdempotency(sweepCtx, db, retention); err != nil {
					l.WithError(err).Warn("idempotency.sweep_failed")
				}
			}
		}
	})
}

package equipslot

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Extend upserts the character's slot extension. It EXTENDS rather than
// duplicates (FR-SLOT-4): the new expiry is max(now, existing) + period, so
// buying it again while it is still active adds to the remaining time instead
// of resetting or creating a second row.
//
// transactionId is the idempotency key of this call (task-240 task 24c): the
// zero UUID means "no dedupe key supplied" (every caller that predates task
// 24c) and always proceeds as before. A non-zero transactionId that already
// matches the row's stored TransactionId means this is a redelivery of a
// call already applied (the outbox atlas-cashshop's purchase queues this
// through is at-least-once) -- it is a no-op that returns the CURRENT
// ExpiresAt without adding days again.
func Extend(db *gorm.DB, tenantId uuid.UUID, characterId uint32, slotIndex int16, period time.Duration, transactionId uuid.UUID) (time.Time, error) {
	var expiresAt time.Time
	err := db.Transaction(func(tx *gorm.DB) error {
		var e Entity
		err := tx.Where("tenant_id = ? AND character_id = ? AND slot_index = ?", tenantId, characterId, slotIndex).First(&e).Error
		now := time.Now()
		base := now
		if err == nil {
			if transactionId != uuid.Nil && e.TransactionId == transactionId {
				expiresAt = e.ExpiresAt
				return nil
			}
			if e.ExpiresAt.After(now) {
				base = e.ExpiresAt
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		expiresAt = base.Add(period)

		ne := &Entity{
			Id:            uuid.New(),
			TenantId:      tenantId,
			CharacterId:   characterId,
			SlotIndex:     slotIndex,
			ExpiresAt:     expiresAt,
			TransactionId: transactionId,
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}, {Name: "slot_index"}},
			DoUpdates: clause.AssignmentColumns([]string{"expires_at", "transaction_id", "updated_at"}),
		}).Create(ne).Error
	})
	if err != nil {
		return time.Time{}, err
	}
	return expiresAt, nil
}

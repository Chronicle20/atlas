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
func Extend(db *gorm.DB, tenantId uuid.UUID, characterId uint32, slotIndex int16, period time.Duration) (time.Time, error) {
	var expiresAt time.Time
	err := db.Transaction(func(tx *gorm.DB) error {
		var e Entity
		err := tx.Where("tenant_id = ? AND character_id = ? AND slot_index = ?", tenantId, characterId, slotIndex).First(&e).Error
		now := time.Now()
		base := now
		if err == nil {
			if e.ExpiresAt.After(now) {
				base = e.ExpiresAt
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		expiresAt = base.Add(period)

		ne := &Entity{
			Id:          uuid.New(),
			TenantId:    tenantId,
			CharacterId: characterId,
			SlotIndex:   slotIndex,
			ExpiresAt:   expiresAt,
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}, {Name: "slot_index"}},
			DoUpdates: clause.AssignmentColumns([]string{"expires_at", "updated_at"}),
		}).Create(ne).Error
	})
	if err != nil {
		return time.Time{}, err
	}
	return expiresAt, nil
}

// GetActive returns the character's currently-active extensions, i.e. those
// whose ExpiresAt is in the future. An expired row is not returned and is not
// deleted -- the history is kept.
func GetActive(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error) {
	var es []Entity
	err := db.Where("tenant_id = ? AND character_id = ? AND expires_at > ?", tenantId, characterId, time.Now()).Find(&es).Error
	if err != nil {
		return nil, err
	}
	ms := make([]Model, 0, len(es))
	for _, e := range es {
		ms = append(ms, modelFromEntity(e))
	}
	return ms, nil
}

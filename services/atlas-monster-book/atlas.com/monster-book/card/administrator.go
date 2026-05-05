package card

import (
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpsertResult struct {
	Inserted  bool
	NewLevel  uint8
	Full      bool
	Duplicate bool
}

// upsertCard inserts at level 1 or increments level (cap MaxLevel) for an existing row,
// guarded by lastEventId for idempotency. Runs in the caller's transaction.
func upsertCard(db *gorm.DB, tenantId uuid.UUID, characterId character.Id, cardId item.Id, eventId uuid.UUID) (UpsertResult, error) {
	if !IsCardId(cardId) {
		return UpsertResult{}, errors.New("cardId is not a monster-book card item")
	}

	// Try to load existing row.
	var existing entity
	err := db.Where("tenant_id = ? AND character_id = ? AND card_id = ?", tenantId, uint32(characterId), uint32(cardId)).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		e := entity{
			TenantId:    tenantId,
			CharacterId: uint32(characterId),
			CardId:      uint32(cardId),
			Level:       1,
			IsSpecial:   IsSpecialCard(cardId),
			LastEventId: &eventId,
		}
		if err := db.Create(&e).Error; err != nil {
			return UpsertResult{}, err
		}
		return UpsertResult{Inserted: true, NewLevel: 1, Full: MaxLevel == 1, Duplicate: false}, nil
	}
	if err != nil {
		return UpsertResult{}, err
	}

	// Idempotency guard: same eventId -> no-op.
	if existing.LastEventId != nil && *existing.LastEventId == eventId {
		return UpsertResult{Inserted: false, NewLevel: existing.Level, Full: existing.Level >= MaxLevel, Duplicate: true}, nil
	}

	if existing.Level >= MaxLevel {
		// Persist the eventId so future replays of *this* eventId no-op,
		// but level stays capped.
		if err := db.Model(&entity{}).
			Where("tenant_id = ? AND character_id = ? AND card_id = ?", tenantId, uint32(characterId), uint32(cardId)).
			Update("last_event_id", eventId).Error; err != nil {
			return UpsertResult{}, err
		}
		return UpsertResult{Inserted: false, NewLevel: MaxLevel, Full: true, Duplicate: false}, nil
	}

	newLevel := existing.Level + 1
	if err := db.Model(&entity{}).
		Where("tenant_id = ? AND character_id = ? AND card_id = ?", tenantId, uint32(characterId), uint32(cardId)).
		Updates(map[string]interface{}{"level": newLevel, "last_event_id": eventId}).Error; err != nil {
		return UpsertResult{}, err
	}
	return UpsertResult{Inserted: false, NewLevel: newLevel, Full: newLevel >= MaxLevel, Duplicate: false}, nil
}

func deleteByCharacter(db *gorm.DB, tenantId uuid.UUID, characterId character.Id) error {
	return db.Where("tenant_id = ? AND character_id = ?", tenantId, uint32(characterId)).Delete(&entity{}).Error
}

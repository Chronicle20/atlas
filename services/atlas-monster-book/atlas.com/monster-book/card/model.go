package card

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
)

const (
	MaxLevel uint8 = 5
)

// IsCardId reports whether the given itemId is a monster-book card item.
func IsCardId(itemId item.Id) bool {
	return item.GetClassification(itemId) == item.ClassificationConsumableMonsterCard
}

// IsSpecialCard reports whether the given cardId belongs to the special-card
// range. The threshold (cardId/1000 >= 2388) is a v1 hardcoded knob — see
// design §6.4.
func IsSpecialCard(cardId item.Id) bool {
	return uint32(cardId)/1000 >= 2388
}

type Model struct {
	tenantId        uuid.UUID
	characterId     character.Id
	cardId          item.Id
	level           uint8
	isSpecial       bool
	lastEventId     *uuid.UUID
	firstAcquiredAt time.Time
	updatedAt       time.Time
}

func (m Model) TenantId() uuid.UUID        { return m.tenantId }
func (m Model) CharacterId() character.Id  { return m.characterId }
func (m Model) CardId() item.Id            { return m.cardId }
func (m Model) Level() uint8               { return m.level }
func (m Model) IsSpecial() bool            { return m.isSpecial }
func (m Model) LastEventId() *uuid.UUID    { return m.lastEventId }
func (m Model) FirstAcquiredAt() time.Time { return m.firstAcquiredAt }
func (m Model) UpdatedAt() time.Time       { return m.updatedAt }

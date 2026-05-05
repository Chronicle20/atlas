package card

import (
	"time"

	"github.com/google/uuid"
)

const (
	MinCardId       uint32 = 2380000
	MaxCardId       uint32 = 2389999
	SpecialCardBase uint32 = 2388000 // cardId/1000 >= 2388
	MaxLevel        uint8  = 5
)

func IsCardId(itemId uint32) bool {
	return itemId >= MinCardId && itemId <= MaxCardId
}

func IsSpecialCard(cardId uint32) bool {
	return cardId/1000 >= 2388
}

type Model struct {
	tenantId        uuid.UUID
	characterId     uint32
	cardId          uint32
	level           uint8
	isSpecial       bool
	lastEventId     *uuid.UUID
	firstAcquiredAt time.Time
	updatedAt       time.Time
}

func (m Model) TenantId() uuid.UUID        { return m.tenantId }
func (m Model) CharacterId() uint32        { return m.characterId }
func (m Model) CardId() uint32             { return m.cardId }
func (m Model) Level() uint8               { return m.level }
func (m Model) IsSpecial() bool            { return m.isSpecial }
func (m Model) LastEventId() *uuid.UUID    { return m.lastEventId }
func (m Model) FirstAcquiredAt() time.Time { return m.firstAcquiredAt }
func (m Model) UpdatedAt() time.Time       { return m.updatedAt }

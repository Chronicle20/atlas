package collection

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
)

type Model struct {
	tenantId         uuid.UUID
	characterId      character.Id
	coverCardId      item.Id
	bookLevel        uint16
	normalCount      uint16
	specialCount     uint16
	expBonusPercent  uint16
	lastCoverEventId *uuid.UUID
	createdAt        time.Time
	updatedAt        time.Time
}

func (m Model) TenantId() uuid.UUID          { return m.tenantId }
func (m Model) CharacterId() character.Id    { return m.characterId }
func (m Model) CoverCardId() item.Id         { return m.coverCardId }
func (m Model) BookLevel() uint16            { return m.bookLevel }
func (m Model) NormalCount() uint16          { return m.normalCount }
func (m Model) SpecialCount() uint16         { return m.specialCount }
func (m Model) ExpBonusPercent() uint16      { return m.expBonusPercent }
func (m Model) LastCoverEventId() *uuid.UUID { return m.lastCoverEventId }
func (m Model) CreatedAt() time.Time         { return m.createdAt }
func (m Model) UpdatedAt() time.Time         { return m.updatedAt }
func (m Model) TotalUniqueCards() uint16     { return m.normalCount + m.specialCount }

// ToEntity is the inverse of Make: it projects the immutable Model into the
// GORM entity used for persistence. CoverCardId is conditionally written by
// callers that intend to update it, but the projection is unconditional —
// upsertStats deliberately omits this method because it composes a partial
// statsUpdate, not a full Model.
func (m Model) ToEntity() entity {
	return entity{
		TenantId:         m.tenantId,
		CharacterId:      uint32(m.characterId),
		CoverCardId:      uint32(m.coverCardId),
		BookLevel:        m.bookLevel,
		NormalCount:      m.normalCount,
		SpecialCount:     m.specialCount,
		ExpBonusPercent:  m.expBonusPercent,
		LastCoverEventId: m.lastCoverEventId,
		CreatedAt:        m.createdAt,
		UpdatedAt:        m.updatedAt,
	}
}

package collection

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	tenantId         uuid.UUID
	characterId      uint32
	coverCardId      uint32
	bookLevel        uint16
	normalCount      uint16
	specialCount     uint16
	expBonusPercent  uint16
	lastCoverEventId *uuid.UUID
	createdAt        time.Time
	updatedAt        time.Time
}

func (m Model) TenantId() uuid.UUID          { return m.tenantId }
func (m Model) CharacterId() uint32          { return m.characterId }
func (m Model) CoverCardId() uint32          { return m.coverCardId }
func (m Model) BookLevel() uint16            { return m.bookLevel }
func (m Model) NormalCount() uint16          { return m.normalCount }
func (m Model) SpecialCount() uint16         { return m.specialCount }
func (m Model) ExpBonusPercent() uint16      { return m.expBonusPercent }
func (m Model) LastCoverEventId() *uuid.UUID { return m.lastCoverEventId }
func (m Model) CreatedAt() time.Time         { return m.createdAt }
func (m Model) UpdatedAt() time.Time         { return m.updatedAt }
func (m Model) TotalUniqueCards() uint16     { return m.normalCount + m.specialCount }

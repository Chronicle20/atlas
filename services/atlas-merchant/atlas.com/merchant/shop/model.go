package shop

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	id           uuid.UUID
	characterId  uint32
	shopType     ShopType
	state        State
	title        string
	mapId        uint32
	x            int16
	y            int16
	permitItemId uint32
	createdAt    time.Time
	expiresAt    *time.Time
	closedAt     *time.Time
	closeReason  CloseReason
	mesoBalance  uint32
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) ShopType() ShopType {
	return m.shopType
}

func (m Model) State() State {
	return m.state
}

func (m Model) Title() string {
	return m.title
}

func (m Model) MapId() uint32 {
	return m.mapId
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) PermitItemId() uint32 {
	return m.permitItemId
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

func (m Model) ExpiresAt() *time.Time {
	return m.expiresAt
}

func (m Model) ClosedAt() *time.Time {
	return m.closedAt
}

func (m Model) CloseReason() CloseReason {
	return m.closeReason
}

func (m Model) MesoBalance() uint32 {
	return m.mesoBalance
}

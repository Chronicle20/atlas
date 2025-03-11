package item

import (
	"github.com/Chronicle20/atlas-constants/item"
)

type Model struct {
	id       uint32
	itemId   uint32
	slot     int16
	quantity uint32
}

func (m Model) Slot() int16 {
	return m.slot
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) Expiration() int64 {
	return -1
}

func (m Model) Owner() string {
	return ""
}

func (m Model) Flag() uint16 {
	return 0
}

func (m Model) Rechargeable() bool {
	return m.ThrowingStar() || m.Bullet()
}

func (m Model) ThrowingStar() bool {
	return item.IsThrowingStar(item.Id(m.ItemId()))
}

func (m Model) Bullet() bool {
	return item.IsBullet(item.Id(m.ItemId()))
}

func (m Model) Id() uint32 {
	return m.id
}

package drop

import (
	"time"
)

type Model struct {
	id           uint32
	itemId       uint32
	equipmentId  uint32
	quantity     uint32
	meso         uint32
	dropType     byte
	x            int16
	y            int16
	ownerId      uint32
	ownerPartyId uint32
	dropTime     time.Time
	dropperId    uint32
	dropperX     int16
	dropperY     int16
	playerDrop   bool
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) Meso() uint32 {
	return m.meso
}

func (m Model) Type() byte {
	return m.dropType
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}

func (m Model) OwnerPartyId() uint32 {
	return m.ownerPartyId
}

func (m Model) DropTime() time.Time {
	return m.dropTime
}

func (m Model) DropperId() uint32 {
	return m.dropperId
}

func (m Model) DropperX() int16 {
	return m.dropperX
}

func (m Model) DropperY() int16 {
	return m.dropperY
}

func (m Model) PlayerDrop() bool {
	return m.playerDrop
}

func (m Model) CharacterDrop() bool {
	return m.playerDrop
}

func (m Model) EquipmentId() uint32 {
	return m.equipmentId
}

type ModelBuilder struct {
	id           uint32
	itemId       uint32
	equipmentId  uint32
	quantity     uint32
	meso         uint32
	dropType     byte
	x            int16
	y            int16
	ownerId      uint32
	ownerPartyId uint32
	dropTime     time.Time
	dropperId    uint32
	dropperX     int16
	dropperY     int16
	playerDrop   bool
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{
		dropTime: time.Now(),
	}
}

func CloneModelBuilder(m Model) *ModelBuilder {
	b := &ModelBuilder{}
	return b.Clone(m)
}

func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.id = id
	return b
}

func (b *ModelBuilder) SetItem(itemId uint32, quantity uint32) *ModelBuilder {
	b.itemId = itemId
	b.quantity = quantity
	return b
}

func (b *ModelBuilder) SetMeso(meso uint32) *ModelBuilder {
	b.meso = meso
	return b
}

func (b *ModelBuilder) SetType(dropType byte) *ModelBuilder {
	b.dropType = dropType
	return b
}

func (b *ModelBuilder) SetEquipmentId(equipmentId uint32) *ModelBuilder {
	b.equipmentId = equipmentId
	return b
}

func (b *ModelBuilder) SetPosition(x int16, y int16) *ModelBuilder {
	b.x = x
	b.y = y
	return b
}

func (b *ModelBuilder) SetOwner(id uint32, partyId uint32) *ModelBuilder {
	b.ownerId = id
	b.ownerPartyId = partyId
	return b
}

func (b *ModelBuilder) SetDropper(id uint32, x int16, y int16) *ModelBuilder {
	b.dropperId = id
	b.dropperX = x
	b.dropperY = y
	return b
}

func (b *ModelBuilder) SetPlayerDrop(is bool) *ModelBuilder {
	b.playerDrop = is
	return b
}

func (b *ModelBuilder) Clone(m Model) *ModelBuilder {
	b.id = m.Id()
	b.itemId = m.ItemId()
	b.equipmentId = m.EquipmentId()
	b.quantity = m.Quantity()
	b.meso = m.Meso()
	b.dropType = m.Type()
	b.x = m.X()
	b.y = m.Y()
	b.ownerId = m.OwnerId()
	b.ownerPartyId = m.OwnerPartyId()
	b.dropTime = m.DropTime()
	b.dropperId = m.DropperId()
	b.dropperX = m.DropperX()
	b.dropperY = m.DropperY()
	b.playerDrop = m.PlayerDrop()
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:           b.id,
		itemId:       b.itemId,
		equipmentId:  b.equipmentId,
		quantity:     b.quantity,
		meso:         b.meso,
		dropType:     b.dropType,
		x:            b.x,
		y:            b.y,
		ownerId:      b.ownerId,
		ownerPartyId: b.ownerPartyId,
		dropTime:     b.dropTime,
		dropperId:    b.dropperId,
		dropperX:     b.dropperX,
		dropperY:     b.dropperY,
		playerDrop:   b.playerDrop,
	}
}

func (m Model) Owner() uint32 {
	if m.ownerPartyId != 0 {
		return m.ownerPartyId
	}
	return m.ownerId
}

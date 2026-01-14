package drop

import (
	"errors"
	"time"
)

var (
	ErrInvalidId = errors.New("drop id must be greater than 0")
)

type modelBuilder struct {
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

// NewModelBuilder creates a new builder instance
func NewModelBuilder() *modelBuilder {
	return &modelBuilder{
		dropTime: time.Now(),
	}
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:           m.id,
		itemId:       m.itemId,
		equipmentId:  m.equipmentId,
		quantity:     m.quantity,
		meso:         m.meso,
		dropType:     m.dropType,
		x:            m.x,
		y:            m.y,
		ownerId:      m.ownerId,
		ownerPartyId: m.ownerPartyId,
		dropTime:     m.dropTime,
		dropperId:    m.dropperId,
		dropperX:     m.dropperX,
		dropperY:     m.dropperY,
		playerDrop:   m.playerDrop,
	}
}

// CloneModelBuilder is an alias for CloneModel for backward compatibility
func CloneModelBuilder(m Model) *modelBuilder {
	return CloneModel(m)
}

func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetItem(itemId uint32, quantity uint32) *modelBuilder {
	b.itemId = itemId
	b.quantity = quantity
	return b
}

func (b *modelBuilder) SetMeso(meso uint32) *modelBuilder {
	b.meso = meso
	return b
}

func (b *modelBuilder) SetType(dropType byte) *modelBuilder {
	b.dropType = dropType
	return b
}

func (b *modelBuilder) SetEquipmentId(equipmentId uint32) *modelBuilder {
	b.equipmentId = equipmentId
	return b
}

func (b *modelBuilder) SetPosition(x int16, y int16) *modelBuilder {
	b.x = x
	b.y = y
	return b
}

func (b *modelBuilder) SetOwner(id uint32, partyId uint32) *modelBuilder {
	b.ownerId = id
	b.ownerPartyId = partyId
	return b
}

func (b *modelBuilder) SetDropper(id uint32, x int16, y int16) *modelBuilder {
	b.dropperId = id
	b.dropperX = x
	b.dropperY = y
	return b
}

func (b *modelBuilder) SetPlayerDrop(is bool) *modelBuilder {
	b.playerDrop = is
	return b
}

// Clone sets the builder's fields from the given model (for chaining)
func (b *modelBuilder) Clone(m Model) *modelBuilder {
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

// Build creates a new Model instance with validation
func (b *modelBuilder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidId
	}
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
	}, nil
}

// MustBuild creates a new Model instance, panicking on validation error
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}

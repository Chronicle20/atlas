package drop

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"time"
)

const (
	StatusAvailable = "AVAILABLE"
	StatusReserved  = "RESERVED"
)

type Model struct {
	tenant        tenant.Model
	id            uint32
	transactionId uuid.UUID
	worldId       world.Id
	channelId     channel.Id
	mapId         _map.Id
	itemId        uint32
	equipmentId   uint32
	quantity      uint32
	meso          uint32
	dropType      byte
	x             int16
	y             int16
	ownerId       uint32
	ownerPartyId  uint32
	dropTime      time.Time
	dropperId     uint32
	dropperX      int16
	dropperY      int16
	playerDrop    bool
	status        string
	petSlot       int8
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

func (m Model) Status() string {
	return m.status
}

func (m Model) CancelReservation() Model {
	return CloneModelBuilder(m).SetStatus(StatusAvailable).SetPetSlot(-1).Build()
}

func (m Model) Reserve(petSlot int8) Model {
	return CloneModelBuilder(m).SetStatus(StatusReserved).SetPetSlot(petSlot).Build()
}

func (m Model) MapId() _map.Id {
	return m.mapId
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) TransactionId() uuid.UUID {
	return m.transactionId
}

func (m Model) CharacterDrop() bool {
	return m.playerDrop
}

func (m Model) EquipmentId() uint32 {
	return m.equipmentId
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) PetSlot() int8 {
	return m.petSlot
}

type ModelBuilder struct {
	tenant        tenant.Model
	id            uint32
	transactionId uuid.UUID
	worldId       world.Id
	channelId     channel.Id
	mapId         _map.Id
	itemId        uint32
	equipmentId   uint32
	quantity      uint32
	meso          uint32
	dropType      byte
	x             int16
	y             int16
	ownerId       uint32
	ownerPartyId  uint32
	dropTime      time.Time
	dropperId     uint32
	dropperX      int16
	dropperY      int16
	playerDrop    bool
	status        string
	petSlot       int8
}

func NewModelBuilder(tenant tenant.Model, worldId world.Id, channelId channel.Id, mapId _map.Id) *ModelBuilder {
	return &ModelBuilder{
		tenant:        tenant,
		transactionId: uuid.New(),
		worldId:       worldId,
		channelId:     channelId,
		mapId:         mapId,
		dropTime:      time.Now(),
		petSlot:       -1,
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

func (b *ModelBuilder) SetTransactionId(transactionId uuid.UUID) *ModelBuilder {
	b.transactionId = transactionId
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

func (b *ModelBuilder) SetStatus(status string) *ModelBuilder {
	b.status = status
	return b
}

func (b *ModelBuilder) SetPetSlot(petSlot int8) *ModelBuilder {
	b.petSlot = petSlot
	return b
}

func (b *ModelBuilder) Clone(m Model) *ModelBuilder {
	b.tenant = m.Tenant()
	b.id = m.Id()
	b.transactionId = m.TransactionId()
	b.worldId = m.WorldId()
	b.channelId = m.ChannelId()
	b.mapId = m.MapId()
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
	b.status = m.Status()
	b.petSlot = m.PetSlot()
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		tenant:        b.tenant,
		id:            b.id,
		transactionId: b.transactionId,
		worldId:       b.worldId,
		channelId:     b.channelId,
		mapId:         b.mapId,
		itemId:        b.itemId,
		equipmentId:   b.equipmentId,
		quantity:      b.quantity,
		meso:          b.meso,
		dropType:      b.dropType,
		x:             b.x,
		y:             b.y,
		ownerId:       b.ownerId,
		ownerPartyId:  b.ownerPartyId,
		dropTime:      b.dropTime,
		dropperId:     b.dropperId,
		dropperX:      b.dropperX,
		dropperY:      b.dropperY,
		playerDrop:    b.playerDrop,
		status:        b.status,
		petSlot:       b.petSlot,
	}
}

func (b *ModelBuilder) ItemId() uint32 {
	return b.itemId
}

func (b *ModelBuilder) WorldId() world.Id {
	return b.worldId
}

func (b *ModelBuilder) ChannelId() channel.Id {
	return b.channelId
}

func (b *ModelBuilder) MapId() _map.Id {
	return b.mapId
}

func (b *ModelBuilder) TransactionId() uuid.UUID {
	return b.transactionId
}

func (b *ModelBuilder) Tenant() tenant.Model {
	return b.tenant
}

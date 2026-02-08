package drop

import (
	"errors"

	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

const (
	StatusAvailable = "AVAILABLE"
	StatusReserved  = "RESERVED"
)

type Model struct {
	tenant        tenant.Model
	id            uint32
	transactionId uuid.UUID
	field         field.Model
	itemId        uint32
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
	// Equipment stats (inline, replacing equipmentId)
	strength      uint16
	dexterity     uint16
	intelligence  uint16
	luck          uint16
	hp            uint16
	mp            uint16
	weaponAttack  uint16
	magicAttack   uint16
	weaponDefense uint16
	magicDefense  uint16
	accuracy      uint16
	avoidability  uint16
	hands         uint16
	speed         uint16
	jump          uint16
	slots         uint16
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
	return CloneModelBuilder(m).SetStatus(StatusAvailable).SetPetSlot(-1).MustBuild()
}

func (m Model) Reserve(petSlot int8) Model {
	return CloneModelBuilder(m).SetStatus(StatusReserved).SetPetSlot(petSlot).MustBuild()
}

func (m Model) Field() field.Model {
	return m.field
}

func (m Model) WorldId() world.Id {
	return m.Field().WorldId()
}

func (m Model) ChannelId() channel.Id {
	return m.Field().ChannelId()
}

func (m Model) MapId() _map.Id {
	return m.Field().MapId()
}

func (m Model) Instance() uuid.UUID {
	return m.Field().Instance()
}

func (m Model) TransactionId() uuid.UUID {
	return m.transactionId
}

func (m Model) CharacterDrop() bool {
	return m.playerDrop
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) PetSlot() int8 {
	return m.petSlot
}

func (m Model) Strength() uint16 {
	return m.strength
}

func (m Model) Dexterity() uint16 {
	return m.dexterity
}

func (m Model) Intelligence() uint16 {
	return m.intelligence
}

func (m Model) Luck() uint16 {
	return m.luck
}

func (m Model) Hp() uint16 {
	return m.hp
}

func (m Model) Mp() uint16 {
	return m.mp
}

func (m Model) WeaponAttack() uint16 {
	return m.weaponAttack
}

func (m Model) MagicAttack() uint16 {
	return m.magicAttack
}

func (m Model) WeaponDefense() uint16 {
	return m.weaponDefense
}

func (m Model) MagicDefense() uint16 {
	return m.magicDefense
}

func (m Model) Accuracy() uint16 {
	return m.accuracy
}

func (m Model) Avoidability() uint16 {
	return m.avoidability
}

func (m Model) Hands() uint16 {
	return m.hands
}

func (m Model) Speed() uint16 {
	return m.speed
}

func (m Model) Jump() uint16 {
	return m.jump
}

func (m Model) Slots() uint16 {
	return m.slots
}

type ModelBuilder struct {
	tenant        tenant.Model
	id            uint32
	transactionId uuid.UUID
	field         field.Model
	itemId        uint32
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
	// Equipment stats
	strength      uint16
	dexterity     uint16
	intelligence  uint16
	luck          uint16
	hp            uint16
	mp            uint16
	weaponAttack  uint16
	magicAttack   uint16
	weaponDefense uint16
	magicDefense  uint16
	accuracy      uint16
	avoidability  uint16
	hands         uint16
	speed         uint16
	jump          uint16
	slots         uint16
}

func NewModelBuilder(tenant tenant.Model, f field.Model) *ModelBuilder {
	return &ModelBuilder{
		tenant:        tenant,
		transactionId: uuid.New(),
		field:         f,
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

func (b *ModelBuilder) SetStrength(v uint16) *ModelBuilder {
	b.strength = v
	return b
}

func (b *ModelBuilder) SetDexterity(v uint16) *ModelBuilder {
	b.dexterity = v
	return b
}

func (b *ModelBuilder) SetIntelligence(v uint16) *ModelBuilder {
	b.intelligence = v
	return b
}

func (b *ModelBuilder) SetLuck(v uint16) *ModelBuilder {
	b.luck = v
	return b
}

func (b *ModelBuilder) SetHp(v uint16) *ModelBuilder {
	b.hp = v
	return b
}

func (b *ModelBuilder) SetMp(v uint16) *ModelBuilder {
	b.mp = v
	return b
}

func (b *ModelBuilder) SetWeaponAttack(v uint16) *ModelBuilder {
	b.weaponAttack = v
	return b
}

func (b *ModelBuilder) SetMagicAttack(v uint16) *ModelBuilder {
	b.magicAttack = v
	return b
}

func (b *ModelBuilder) SetWeaponDefense(v uint16) *ModelBuilder {
	b.weaponDefense = v
	return b
}

func (b *ModelBuilder) SetMagicDefense(v uint16) *ModelBuilder {
	b.magicDefense = v
	return b
}

func (b *ModelBuilder) SetAccuracy(v uint16) *ModelBuilder {
	b.accuracy = v
	return b
}

func (b *ModelBuilder) SetAvoidability(v uint16) *ModelBuilder {
	b.avoidability = v
	return b
}

func (b *ModelBuilder) SetHands(v uint16) *ModelBuilder {
	b.hands = v
	return b
}

func (b *ModelBuilder) SetSpeed(v uint16) *ModelBuilder {
	b.speed = v
	return b
}

func (b *ModelBuilder) SetJump(v uint16) *ModelBuilder {
	b.jump = v
	return b
}

func (b *ModelBuilder) SetSlots(v uint16) *ModelBuilder {
	b.slots = v
	return b
}

func (b *ModelBuilder) Clone(m Model) *ModelBuilder {
	b.tenant = m.Tenant()
	b.id = m.Id()
	b.transactionId = m.TransactionId()
	b.field = m.Field()
	b.itemId = m.ItemId()
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
	b.strength = m.Strength()
	b.dexterity = m.Dexterity()
	b.intelligence = m.Intelligence()
	b.luck = m.Luck()
	b.hp = m.Hp()
	b.mp = m.Mp()
	b.weaponAttack = m.WeaponAttack()
	b.magicAttack = m.MagicAttack()
	b.weaponDefense = m.WeaponDefense()
	b.magicDefense = m.MagicDefense()
	b.accuracy = m.Accuracy()
	b.avoidability = m.Avoidability()
	b.hands = m.Hands()
	b.speed = m.Speed()
	b.jump = m.Jump()
	b.slots = m.Slots()
	return b
}

func (b *ModelBuilder) Build() (Model, error) {
	if b.tenant.Id() == uuid.Nil {
		return Model{}, errors.New("tenant is required")
	}
	if b.transactionId == uuid.Nil {
		return Model{}, errors.New("transactionId is required")
	}
	return Model{
		tenant:        b.tenant,
		id:            b.id,
		transactionId: b.transactionId,
		field:         b.field,
		itemId:        b.itemId,
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
		strength:      b.strength,
		dexterity:     b.dexterity,
		intelligence:  b.intelligence,
		luck:          b.luck,
		hp:            b.hp,
		mp:            b.mp,
		weaponAttack:  b.weaponAttack,
		magicAttack:   b.magicAttack,
		weaponDefense: b.weaponDefense,
		magicDefense:  b.magicDefense,
		accuracy:      b.accuracy,
		avoidability:  b.avoidability,
		hands:         b.hands,
		speed:         b.speed,
		jump:          b.jump,
		slots:         b.slots,
	}, nil
}

// MustBuild builds the model and panics if validation fails.
// Use this only when building from a known-valid source (e.g., cloning an existing model).
func (b *ModelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild failed: " + err.Error())
	}
	return m
}

func (b *ModelBuilder) ItemId() uint32 {
	return b.itemId
}

func (b *ModelBuilder) Field() field.Model {
	return b.field
}

func (b *ModelBuilder) TransactionId() uuid.UUID {
	return b.transactionId
}

func (b *ModelBuilder) Tenant() tenant.Model {
	return b.tenant
}

package drop

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Builder struct {
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

func NewBuilder(tenant tenant.Model, f field.Model) *Builder {
	return &Builder{
		tenant:        tenant,
		transactionId: uuid.New(),
		field:         f,
		dropTime:      time.Now(),
		petSlot:       -1,
	}
}

func CloneBuilder(m Model) *Builder {
	b := &Builder{}
	return b.Clone(m)
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetTransactionId(transactionId uuid.UUID) *Builder {
	b.transactionId = transactionId
	return b
}

func (b *Builder) SetItem(itemId uint32, quantity uint32) *Builder {
	b.itemId = itemId
	b.quantity = quantity
	return b
}

func (b *Builder) SetMeso(meso uint32) *Builder {
	b.meso = meso
	return b
}

func (b *Builder) SetType(dropType byte) *Builder {
	b.dropType = dropType
	return b
}

func (b *Builder) SetPosition(x int16, y int16) *Builder {
	b.x = x
	b.y = y
	return b
}

func (b *Builder) SetOwner(id uint32, partyId uint32) *Builder {
	b.ownerId = id
	b.ownerPartyId = partyId
	return b
}

func (b *Builder) SetDropper(id uint32, x int16, y int16) *Builder {
	b.dropperId = id
	b.dropperX = x
	b.dropperY = y
	return b
}

func (b *Builder) SetPlayerDrop(is bool) *Builder {
	b.playerDrop = is
	return b
}

func (b *Builder) SetStatus(status string) *Builder {
	b.status = status
	return b
}

func (b *Builder) SetPetSlot(petSlot int8) *Builder {
	b.petSlot = petSlot
	return b
}

func (b *Builder) SetStrength(v uint16) *Builder {
	b.strength = v
	return b
}

func (b *Builder) SetDexterity(v uint16) *Builder {
	b.dexterity = v
	return b
}

func (b *Builder) SetIntelligence(v uint16) *Builder {
	b.intelligence = v
	return b
}

func (b *Builder) SetLuck(v uint16) *Builder {
	b.luck = v
	return b
}

func (b *Builder) SetHp(v uint16) *Builder {
	b.hp = v
	return b
}

func (b *Builder) SetMp(v uint16) *Builder {
	b.mp = v
	return b
}

func (b *Builder) SetWeaponAttack(v uint16) *Builder {
	b.weaponAttack = v
	return b
}

func (b *Builder) SetMagicAttack(v uint16) *Builder {
	b.magicAttack = v
	return b
}

func (b *Builder) SetWeaponDefense(v uint16) *Builder {
	b.weaponDefense = v
	return b
}

func (b *Builder) SetMagicDefense(v uint16) *Builder {
	b.magicDefense = v
	return b
}

func (b *Builder) SetAccuracy(v uint16) *Builder {
	b.accuracy = v
	return b
}

func (b *Builder) SetAvoidability(v uint16) *Builder {
	b.avoidability = v
	return b
}

func (b *Builder) SetHands(v uint16) *Builder {
	b.hands = v
	return b
}

func (b *Builder) SetSpeed(v uint16) *Builder {
	b.speed = v
	return b
}

func (b *Builder) SetJump(v uint16) *Builder {
	b.jump = v
	return b
}

func (b *Builder) SetSlots(v uint16) *Builder {
	b.slots = v
	return b
}

func (b *Builder) Clone(m Model) *Builder {
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

func (b *Builder) Build() (Model, error) {
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
func (b *Builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild failed: " + err.Error())
	}
	return m
}

func (b *Builder) ItemId() uint32 {
	return b.itemId
}

func (b *Builder) Field() field.Model {
	return b.field
}

func (b *Builder) TransactionId() uuid.UUID {
	return b.transactionId
}

func (b *Builder) Tenant() tenant.Model {
	return b.tenant
}

package holding

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// Builder constructs an immutable holding Model. The id is assigned at create
// time in the administrator, so it is not required here.
type Builder struct {
	id       uuid.UUID
	tenantId uuid.UUID
	worldId  world.Id
	serial   uint32
	ownerId  uint32

	origin Origin

	templateId uint32
	quantity   uint32

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
	level         byte
	itemLevel     byte
	itemExp       uint32
	ringId        uint32
	viciousCount  uint32
	flags         uint16

	createdAt time.Time
}

func NewBuilder(tenantId uuid.UUID, worldId world.Id, ownerId uint32) *Builder {
	return &Builder{tenantId: tenantId, worldId: worldId, ownerId: ownerId}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetSerial(serial uint32) *Builder {
	b.serial = serial
	return b
}

func (b *Builder) SetOwnerId(ownerId uint32) *Builder {
	b.ownerId = ownerId
	return b
}

func (b *Builder) SetOrigin(o Origin) *Builder {
	b.origin = o
	return b
}

func (b *Builder) SetTemplateId(templateId uint32) *Builder {
	b.templateId = templateId
	return b
}

func (b *Builder) SetQuantity(quantity uint32) *Builder {
	b.quantity = quantity
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

func (b *Builder) SetHP(v uint16) *Builder {
	b.hp = v
	return b
}

func (b *Builder) SetMP(v uint16) *Builder {
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

func (b *Builder) SetLevel(v byte) *Builder {
	b.level = v
	return b
}

func (b *Builder) SetItemLevel(v byte) *Builder {
	b.itemLevel = v
	return b
}

func (b *Builder) SetItemExp(v uint32) *Builder {
	b.itemExp = v
	return b
}

func (b *Builder) SetRingId(v uint32) *Builder {
	b.ringId = v
	return b
}

func (b *Builder) SetViciousCount(v uint32) *Builder {
	b.viciousCount = v
	return b
}

func (b *Builder) SetFlags(v uint16) *Builder {
	b.flags = v
	return b
}

func (b *Builder) SetCreatedAt(v time.Time) *Builder {
	b.createdAt = v
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	return Model{
		id:            b.id,
		tenantId:      b.tenantId,
		worldId:       b.worldId,
		serial:        b.serial,
		ownerId:       b.ownerId,
		origin:        b.origin,
		templateId:    b.templateId,
		quantity:      b.quantity,
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
		level:         b.level,
		itemLevel:     b.itemLevel,
		itemExp:       b.itemExp,
		ringId:        b.ringId,
		viciousCount:  b.viciousCount,
		flags:         b.flags,
		createdAt:     b.createdAt,
	}, nil
}

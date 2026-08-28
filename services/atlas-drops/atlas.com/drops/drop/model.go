package drop

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	StatusAvailable = "AVAILABLE"
	StatusReserved  = "RESERVED"

	OwnershipDuration = 15 * time.Second
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

func (m Model) CanBeReservedBy(characterId uint32, partyId uint32) bool {
	if m.playerDrop {
		return true
	}
	if m.ownerId == 0 && m.ownerPartyId == 0 {
		return true
	}
	if time.Since(m.dropTime) >= OwnershipDuration {
		return true
	}
	if m.ownerId != 0 && m.ownerId == characterId {
		return true
	}
	if m.ownerPartyId != 0 && m.ownerPartyId == partyId {
		return true
	}
	return false
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
	return CloneBuilder(m).SetStatus(StatusAvailable).SetPetSlot(-1).MustBuild()
}

func (m Model) Reserve(petSlot int8) Model {
	return CloneBuilder(m).SetStatus(StatusReserved).SetPetSlot(petSlot).MustBuild()
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

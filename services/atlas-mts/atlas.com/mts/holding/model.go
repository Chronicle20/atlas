package holding

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Origin enumerates why an item entered a character's holding (take-home) bucket.
type Origin string

const (
	OriginPurchased Origin = "purchased"
	OriginUnsold    Origin = "unsold"
	OriginCancelled Origin = "cancelled"
	OriginExpired   Origin = "expired"
)

// Model is the immutable per-world holding: an item awaiting take-home by its
// owner. It carries the same explicit item snapshot as a Listing (template id,
// quantity, and the full equip stat block) plus the origin that placed it here.
// Construct it via the Builder.
type Model struct {
	id       uuid.UUID
	tenantId uuid.UUID
	worldId  world.Id
	serial   uint32
	ownerId  uint32

	origin Origin

	// item snapshot
	templateId uint32
	quantity   uint32

	// equip stat block
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

func (m Model) Id() uuid.UUID         { return m.id }
func (m Model) TenantId() uuid.UUID   { return m.tenantId }
func (m Model) WorldId() world.Id     { return m.worldId }
func (m Model) Serial() uint32        { return m.serial }
func (m Model) OwnerId() uint32       { return m.ownerId }
func (m Model) Origin() Origin        { return m.origin }
func (m Model) TemplateId() uint32    { return m.templateId }
func (m Model) Quantity() uint32      { return m.quantity }
func (m Model) Strength() uint16      { return m.strength }
func (m Model) Dexterity() uint16     { return m.dexterity }
func (m Model) Intelligence() uint16  { return m.intelligence }
func (m Model) Luck() uint16          { return m.luck }
func (m Model) HP() uint16            { return m.hp }
func (m Model) MP() uint16            { return m.mp }
func (m Model) WeaponAttack() uint16  { return m.weaponAttack }
func (m Model) MagicAttack() uint16   { return m.magicAttack }
func (m Model) WeaponDefense() uint16 { return m.weaponDefense }
func (m Model) MagicDefense() uint16  { return m.magicDefense }
func (m Model) Accuracy() uint16      { return m.accuracy }
func (m Model) Avoidability() uint16  { return m.avoidability }
func (m Model) Hands() uint16         { return m.hands }
func (m Model) Speed() uint16         { return m.speed }
func (m Model) Jump() uint16          { return m.jump }
func (m Model) Slots() uint16         { return m.slots }
func (m Model) Level() byte           { return m.level }
func (m Model) ItemLevel() byte       { return m.itemLevel }
func (m Model) ItemExp() uint32       { return m.itemExp }
func (m Model) RingId() uint32        { return m.ringId }
func (m Model) ViciousCount() uint32  { return m.viciousCount }
func (m Model) Flags() uint16         { return m.flags }
func (m Model) CreatedAt() time.Time  { return m.createdAt }

package account

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	tenantId    uuid.UUID
	id          uint32
	name        string
	password    string
	pin         string
	pic         string
	birthDate   uint32
	pinAttempts int
	picAttempts int
	state       State
	gender      byte
	tos         bool
	updatedAt   time.Time
}

func (a Model) Id() uint32 {
	return a.id
}

func (a Model) Name() string {
	return a.name
}

func (a Model) Password() string {
	return a.password
}

func (a Model) State() State {
	return a.state
}

func (a Model) TOS() bool {
	return a.tos
}

func (a Model) UpdatedAt() time.Time {
	return a.updatedAt
}

func (a Model) TenantId() uuid.UUID {
	return a.tenantId
}

func (a Model) Pin() string {
	return a.pin
}

func (a Model) Pic() string {
	return a.pic
}

func (a Model) BirthDate() uint32 {
	return a.birthDate
}

func (a Model) PinAttempts() int {
	return a.pinAttempts
}

func (a Model) PicAttempts() int {
	return a.picAttempts
}

func (a Model) Gender() byte {
	return a.gender
}

func LoggedIn(m Model) bool {
	return m.state != StateNotLoggedIn
}

// CharacterSlotModel is the per-(tenant, account, world) character-slot
// count (task-246 bug-b-type-must-add-a-slot.md).
type CharacterSlotModel struct {
	tenantId  uuid.UUID
	accountId uint32
	worldId   byte
	slots     int16
}

func (m CharacterSlotModel) TenantId() uuid.UUID {
	return m.tenantId
}

func (m CharacterSlotModel) AccountId() uint32 {
	return m.accountId
}

func (m CharacterSlotModel) WorldId() byte {
	return m.worldId
}

func (m CharacterSlotModel) Slots() int16 {
	return m.slots
}

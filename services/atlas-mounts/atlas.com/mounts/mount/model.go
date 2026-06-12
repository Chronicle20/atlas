package mount

import (
	"time"

	"github.com/google/uuid"
)

// Model is an immutable representation of a character's mount progression.
//
// Construct via NewModelBuilder / Clone; mutate via the builder's setters and
// Build(). Fields are private with read-only getters to preserve immutability.
type Model struct {
	tenantId            uuid.UUID
	characterId         uint32
	id                  uuid.UUID
	level               int
	exp                 int
	tiredness           int
	lastTirednessTickAt *time.Time
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) Level() int {
	return m.level
}

func (m Model) Exp() int {
	return m.exp
}

func (m Model) Tiredness() int {
	return m.tiredness
}

func (m Model) LastTirednessTickAt() *time.Time {
	return m.lastTirednessTickAt
}

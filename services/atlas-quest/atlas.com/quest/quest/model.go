package quest

import (
	"atlas-quest/quest/progress"
	"time"

	"github.com/google/uuid"
)

type Model struct {
	tenantId       uuid.UUID
	id             uint32
	characterId    uint32
	questId        uint32
	state          State
	startedAt      time.Time
	completedAt    time.Time
	expirationTime time.Time
	completedCount uint32
	forfeitCount   uint32
	progress       []progress.Model
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) QuestId() uint32 {
	return m.questId
}

func (m Model) State() State {
	return m.state
}

func (m Model) StartedAt() time.Time {
	return m.startedAt
}

func (m Model) CompletedAt() time.Time {
	return m.completedAt
}

func (m Model) ExpirationTime() time.Time {
	return m.expirationTime
}

func (m Model) CompletedCount() uint32 {
	return m.completedCount
}

func (m Model) ForfeitCount() uint32 {
	return m.forfeitCount
}

func (m Model) IsExpired() bool {
	if m.expirationTime.IsZero() {
		return false
	}
	return time.Now().After(m.expirationTime)
}

func (m Model) Progress() []progress.Model {
	return m.progress
}

func (m Model) GetProgress(infoNumber uint32) (progress.Model, bool) {
	for _, p := range m.progress {
		if p.InfoNumber() == infoNumber {
			return p, true
		}
	}
	return progress.Model{}, false
}

package quest

import (
	"atlas-quest/quest/progress"
	"time"

	"github.com/google/uuid"
)

type modelBuilder struct {
	tenantId    uuid.UUID
	id          uint32
	characterId uint32
	questId     uint32
	state       State
	startedAt   time.Time
	completedAt time.Time
	progress    []progress.Model
}

func NewModelBuilder() *modelBuilder {
	return &modelBuilder{
		progress: make([]progress.Model, 0),
	}
}

func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		tenantId:    m.tenantId,
		id:          m.id,
		characterId: m.characterId,
		questId:     m.questId,
		state:       m.state,
		startedAt:   m.startedAt,
		completedAt: m.completedAt,
		progress:    m.progress,
	}
}

func (b *modelBuilder) SetTenantId(tenantId uuid.UUID) *modelBuilder {
	b.tenantId = tenantId
	return b
}

func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetCharacterId(characterId uint32) *modelBuilder {
	b.characterId = characterId
	return b
}

func (b *modelBuilder) SetQuestId(questId uint32) *modelBuilder {
	b.questId = questId
	return b
}

func (b *modelBuilder) SetState(state State) *modelBuilder {
	b.state = state
	return b
}

func (b *modelBuilder) SetStartedAt(startedAt time.Time) *modelBuilder {
	b.startedAt = startedAt
	return b
}

func (b *modelBuilder) SetCompletedAt(completedAt time.Time) *modelBuilder {
	b.completedAt = completedAt
	return b
}

func (b *modelBuilder) SetProgress(progress []progress.Model) *modelBuilder {
	b.progress = progress
	return b
}

func (b *modelBuilder) Build() Model {
	return Model{
		tenantId:    b.tenantId,
		id:          b.id,
		characterId: b.characterId,
		questId:     b.questId,
		state:       b.state,
		startedAt:   b.startedAt,
		completedAt: b.completedAt,
		progress:    b.progress,
	}
}

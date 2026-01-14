package quest

import (
	"atlas-quest/quest/progress"
	"time"

	"github.com/google/uuid"
)

type entityBuilder struct {
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
	progress       []progress.Entity
}

func NewEntityBuilder() *entityBuilder {
	return &entityBuilder{}
}

func CloneEntity(e Entity) *entityBuilder {
	return &entityBuilder{
		tenantId:       e.TenantId,
		id:             e.ID,
		characterId:    e.CharacterId,
		questId:        e.QuestId,
		state:          e.State,
		startedAt:      e.StartedAt,
		completedAt:    e.CompletedAt,
		expirationTime: e.ExpirationTime,
		completedCount: e.CompletedCount,
		forfeitCount:   e.ForfeitCount,
		progress:       e.Progress,
	}
}

func (b *entityBuilder) SetTenantId(tenantId uuid.UUID) *entityBuilder {
	b.tenantId = tenantId
	return b
}

func (b *entityBuilder) SetId(id uint32) *entityBuilder {
	b.id = id
	return b
}

func (b *entityBuilder) SetCharacterId(characterId uint32) *entityBuilder {
	b.characterId = characterId
	return b
}

func (b *entityBuilder) SetQuestId(questId uint32) *entityBuilder {
	b.questId = questId
	return b
}

func (b *entityBuilder) SetState(state State) *entityBuilder {
	b.state = state
	return b
}

func (b *entityBuilder) SetStartedAt(startedAt time.Time) *entityBuilder {
	b.startedAt = startedAt
	return b
}

func (b *entityBuilder) SetCompletedAt(completedAt time.Time) *entityBuilder {
	b.completedAt = completedAt
	return b
}

func (b *entityBuilder) SetExpirationTime(expirationTime time.Time) *entityBuilder {
	b.expirationTime = expirationTime
	return b
}

func (b *entityBuilder) SetCompletedCount(completedCount uint32) *entityBuilder {
	b.completedCount = completedCount
	return b
}

func (b *entityBuilder) SetForfeitCount(forfeitCount uint32) *entityBuilder {
	b.forfeitCount = forfeitCount
	return b
}

func (b *entityBuilder) SetProgress(progress []progress.Entity) *entityBuilder {
	b.progress = progress
	return b
}

func (b *entityBuilder) Build() Entity {
	return Entity{
		TenantId:       b.tenantId,
		ID:             b.id,
		CharacterId:    b.characterId,
		QuestId:        b.questId,
		State:          b.state,
		StartedAt:      b.startedAt,
		CompletedAt:    b.completedAt,
		ExpirationTime: b.expirationTime,
		CompletedCount: b.completedCount,
		ForfeitCount:   b.forfeitCount,
		Progress:       b.progress,
	}
}

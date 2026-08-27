package quest

import (
	"atlas-quest/quest/progress"
	"errors"
	"time"

	"github.com/google/uuid"
)

type builder struct {
	tenantId    uuid.UUID
	id          uint32
	characterId uint32
	questId     uint32
	state       State
	startedAt   time.Time
	completedAt time.Time
	progress    []progress.Model
}

func NewBuilder() *builder {
	return &builder{
		progress: make([]progress.Model, 0),
	}
}

func CloneModel(m Model) *builder {
	return &builder{
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

func (b *builder) SetTenantId(tenantId uuid.UUID) *builder {
	b.tenantId = tenantId
	return b
}

func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

func (b *builder) SetCharacterId(characterId uint32) *builder {
	b.characterId = characterId
	return b
}

func (b *builder) SetQuestId(questId uint32) *builder {
	b.questId = questId
	return b
}

func (b *builder) SetState(state State) *builder {
	b.state = state
	return b
}

func (b *builder) SetStartedAt(startedAt time.Time) *builder {
	b.startedAt = startedAt
	return b
}

func (b *builder) SetCompletedAt(completedAt time.Time) *builder {
	b.completedAt = completedAt
	return b
}

func (b *builder) SetProgress(progress []progress.Model) *builder {
	b.progress = progress
	return b
}

func (b *builder) Build() Model {
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

// BuildWithValidation returns the built Model with validation, returning an error if required fields are missing.
// This is the recommended method for new code.
func (b *builder) BuildWithValidation() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, ErrMissingTenantId
	}
	if b.characterId == 0 {
		return Model{}, ErrMissingCharacterId
	}
	if b.questId == 0 {
		return Model{}, ErrMissingQuestId
	}
	return b.Build(), nil
}

// Validation errors for builder
var (
	ErrMissingTenantId    = errors.New("tenant ID is required")
	ErrMissingCharacterId = errors.New("character ID is required")
	ErrMissingQuestId     = errors.New("quest ID is required")
)

package occurrence

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Builder provides fluent construction of occurrence models.
type Builder struct {
	id               uuid.UUID
	definitionId     uuid.UUID
	theType          string
	state            string
	stage            string
	context          json.RawMessage
	worldId          world.Id
	channelId        channel.Id
	voyageId         uuid.UUID
	concurrencyKey   string
	startedAt        time.Time
	nextTransitionAt *time.Time
	completedAt      *time.Time
	completionReason string
}

// NewBuilder creates a new builder with the required parameters. State
// defaults to StateActive and StartedAt to time.Now() — every occurrence is
// born active and running.
func NewBuilder(definitionId uuid.UUID, theType string) *Builder {
	return &Builder{
		id:           uuid.New(),
		definitionId: definitionId,
		theType:      theType,
		state:        StateActive,
		startedAt:    time.Now(),
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetState(state string) *Builder {
	b.state = state
	return b
}

func (b *Builder) SetStage(stage string) *Builder {
	b.stage = stage
	return b
}

func (b *Builder) SetContext(context json.RawMessage) *Builder {
	b.context = context
	return b
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetChannelId(channelId channel.Id) *Builder {
	b.channelId = channelId
	return b
}

func (b *Builder) SetVoyageId(voyageId uuid.UUID) *Builder {
	b.voyageId = voyageId
	return b
}

func (b *Builder) SetConcurrencyKey(concurrencyKey string) *Builder {
	b.concurrencyKey = concurrencyKey
	return b
}

func (b *Builder) SetStartedAt(startedAt time.Time) *Builder {
	b.startedAt = startedAt
	return b
}

func (b *Builder) SetNextTransitionAt(nextTransitionAt *time.Time) *Builder {
	b.nextTransitionAt = nextTransitionAt
	return b
}

func (b *Builder) SetCompletedAt(completedAt *time.Time) *Builder {
	b.completedAt = completedAt
	return b
}

func (b *Builder) SetCompletionReason(completionReason string) *Builder {
	b.completionReason = completionReason
	return b
}

// Build validates invariants and constructs the final immutable model.
func (b *Builder) Build() (Model, error) {
	if b.definitionId == uuid.Nil {
		return Model{}, errors.New("definitionId is required")
	}
	if b.theType == "" {
		return Model{}, errors.New("type is required")
	}
	if b.state == "" {
		return Model{}, errors.New("state is required")
	}

	return Model{
		id:               b.id,
		definitionId:     b.definitionId,
		theType:          b.theType,
		state:            b.state,
		stage:            b.stage,
		context:          b.context,
		worldId:          b.worldId,
		channelId:        b.channelId,
		voyageId:         b.voyageId,
		concurrencyKey:   b.concurrencyKey,
		startedAt:        b.startedAt,
		nextTransitionAt: b.nextTransitionAt,
		completedAt:      b.completedAt,
		completionReason: b.completionReason,
	}, nil
}

// Builder returns a builder initialized with the current model's values.
func (m Model) Builder() *Builder {
	return &Builder{
		id:               m.id,
		definitionId:     m.definitionId,
		theType:          m.theType,
		state:            m.state,
		stage:            m.stage,
		context:          m.context,
		worldId:          m.worldId,
		channelId:        m.channelId,
		voyageId:         m.voyageId,
		concurrencyKey:   m.concurrencyKey,
		startedAt:        m.startedAt,
		nextTransitionAt: m.nextTransitionAt,
		completedAt:      m.completedAt,
		completionReason: m.completionReason,
	}
}

package instance

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Builder struct {
	id                uuid.UUID
	tenantId          uuid.UUID
	definitionId      uuid.UUID
	questId           string
	state             State
	worldId           world.Id
	channelId         channel.Id
	partyId           uint32
	characters        []CharacterEntry
	currentStageIndex uint32
	startedAt         time.Time
	stageStartedAt    time.Time
	registeredAt      time.Time
	fieldInstances    []uuid.UUID
	stageState        StageState
	affinityId        uint32
}

func NewBuilder() *Builder {
	return &Builder{
		id:             uuid.New(),
		state:          StateRegistering,
		characters:     make([]CharacterEntry, 0),
		fieldInstances: make([]uuid.UUID, 0),
		stageState:     NewStageState(),
		registeredAt:   time.Now(),
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder           { b.id = id; return b }
func (b *Builder) SetTenantId(id uuid.UUID) *Builder      { b.tenantId = id; return b }
func (b *Builder) SetDefinitionId(id uuid.UUID) *Builder   { b.definitionId = id; return b }
func (b *Builder) SetQuestId(qid string) *Builder          { b.questId = qid; return b }
func (b *Builder) SetState(s State) *Builder               { b.state = s; return b }
func (b *Builder) SetWorldId(wid world.Id) *Builder        { b.worldId = wid; return b }
func (b *Builder) SetChannelId(cid channel.Id) *Builder    { b.channelId = cid; return b }
func (b *Builder) SetPartyId(pid uint32) *Builder          { b.partyId = pid; return b }
func (b *Builder) SetCharacters(c []CharacterEntry) *Builder { b.characters = c; return b }
func (b *Builder) SetAffinityId(id uint32) *Builder           { b.affinityId = id; return b }

func (b *Builder) Build() (Model, error) {
	if b.questId == "" {
		return Model{}, errors.New("questId is required")
	}
	return Model{
		id:                b.id,
		tenantId:          b.tenantId,
		definitionId:      b.definitionId,
		questId:           b.questId,
		state:             b.state,
		worldId:           b.worldId,
		channelId:         b.channelId,
		partyId:           b.partyId,
		characters:        b.characters,
		currentStageIndex: b.currentStageIndex,
		startedAt:         b.startedAt,
		stageStartedAt:    b.stageStartedAt,
		registeredAt:      b.registeredAt,
		fieldInstances:    b.fieldInstances,
		stageState:        b.stageState,
		affinityId:        b.affinityId,
	}, nil
}

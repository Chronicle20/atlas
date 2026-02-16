package instance

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type State string

const (
	StateRegistering State = "registering"
	StateActive      State = "active"
	StateClearing    State = "clearing"
	StateCompleted   State = "completed"
	StateFailed      State = "failed"
)

type CharacterEntry struct {
	CharacterId uint32
	WorldId     world.Id
	ChannelId   channel.Id
}

type StageState struct {
	ItemCounts   map[uint32]uint32
	MonsterKills map[uint32]uint32
	Combination  []uint32
	Attempts     uint32
	CustomData   map[string]any
}

func NewStageState() StageState {
	return StageState{
		ItemCounts:   make(map[uint32]uint32),
		MonsterKills: make(map[uint32]uint32),
		Combination:  make([]uint32, 0),
		CustomData:   make(map[string]any),
	}
}

type Model struct {
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

func (m Model) Id() uuid.UUID                  { return m.id }
func (m Model) TenantId() uuid.UUID            { return m.tenantId }
func (m Model) DefinitionId() uuid.UUID        { return m.definitionId }
func (m Model) QuestId() string                { return m.questId }
func (m Model) State() State                   { return m.state }
func (m Model) WorldId() world.Id              { return m.worldId }
func (m Model) ChannelId() channel.Id          { return m.channelId }
func (m Model) PartyId() uint32                { return m.partyId }
func (m Model) Characters() []CharacterEntry   { return m.characters }
func (m Model) CurrentStageIndex() uint32      { return m.currentStageIndex }
func (m Model) StartedAt() time.Time           { return m.startedAt }
func (m Model) StageStartedAt() time.Time      { return m.stageStartedAt }
func (m Model) RegisteredAt() time.Time        { return m.registeredAt }
func (m Model) FieldInstances() []uuid.UUID    { return m.fieldInstances }
func (m Model) StageState() StageState         { return m.stageState }
func (m Model) AffinityId() uint32             { return m.affinityId }

func (m Model) SetState(s State) Model {
	m.state = s
	return m
}

func (m Model) SetCurrentStageIndex(idx uint32) Model {
	m.currentStageIndex = idx
	return m
}

func (m Model) SetStartedAt(t time.Time) Model {
	m.startedAt = t
	return m
}

func (m Model) SetStageStartedAt(t time.Time) Model {
	m.stageStartedAt = t
	return m
}

func (m Model) SetFieldInstances(fis []uuid.UUID) Model {
	m.fieldInstances = fis
	return m
}

func (m Model) SetStageState(ss StageState) Model {
	m.stageState = ss
	return m
}

func (m Model) SetAffinityId(id uint32) Model {
	m.affinityId = id
	return m
}

func (m Model) AddCharacter(entry CharacterEntry) Model {
	m.characters = append(m.characters, entry)
	return m
}

func (m Model) RemoveCharacter(characterId uint32) Model {
	chars := make([]CharacterEntry, 0, len(m.characters))
	for _, c := range m.characters {
		if c.CharacterId != characterId {
			chars = append(chars, c)
		}
	}
	m.characters = chars
	return m
}

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

func (b *Builder) Build() Model {
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
	}
}

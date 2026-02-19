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
	StateBonus       State = "bonus"
	StateFailed      State = "failed"
)

type CharacterEntry struct {
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
}

func NewCharacterEntry(characterId uint32, worldId world.Id, channelId channel.Id) CharacterEntry {
	return CharacterEntry{
		characterId: characterId,
		worldId:     worldId,
		channelId:   channelId,
	}
}

func (c CharacterEntry) CharacterId() uint32     { return c.characterId }
func (c CharacterEntry) WorldId() world.Id       { return c.worldId }
func (c CharacterEntry) ChannelId() channel.Id   { return c.channelId }

type StageState struct {
	itemCounts   map[uint32]uint32
	monsterKills map[uint32]uint32
	combination  []uint32
	attempts     uint32
	customData   map[string]any
}

func NewStageState() StageState {
	return StageState{
		itemCounts:   make(map[uint32]uint32),
		monsterKills: make(map[uint32]uint32),
		combination:  make([]uint32, 0),
		customData:   make(map[string]any),
	}
}

func (s StageState) ItemCounts() map[uint32]uint32   { return s.itemCounts }
func (s StageState) MonsterKills() map[uint32]uint32  { return s.monsterKills }
func (s StageState) Combination() []uint32            { return s.combination }
func (s StageState) Attempts() uint32                 { return s.attempts }
func (s StageState) CustomData() map[string]any       { return s.customData }

func (s StageState) WithCombination(combo []uint32) StageState {
	s.combination = combo
	return s
}

func (s StageState) WithItemCount(id uint32, delta uint32) StageState {
	counts := make(map[uint32]uint32, len(s.itemCounts))
	for k, v := range s.itemCounts {
		counts[k] = v
	}
	counts[id] += delta
	s.itemCounts = counts
	return s
}

func (s StageState) WithMonsterKill(id uint32, delta uint32) StageState {
	kills := make(map[uint32]uint32, len(s.monsterKills))
	for k, v := range s.monsterKills {
		kills[k] = v
	}
	kills[id] += delta
	s.monsterKills = kills
	return s
}

func (s StageState) WithCustomData(key string, value any) StageState {
	data := make(map[string]any, len(s.customData))
	for k, v := range s.customData {
		data[k] = v
	}
	data[key] = value
	s.customData = data
	return s
}

func (s StageState) IncrementCustomData(key string) StageState {
	data := make(map[string]any, len(s.customData))
	for k, v := range s.customData {
		data[k] = v
	}
	current := 0
	if v, ok := data[key]; ok {
		switch n := v.(type) {
		case float64:
			current = int(n)
		case int:
			current = n
		}
	}
	data[key] = current + 1
	s.customData = data
	return s
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
		if c.characterId != characterId {
			chars = append(chars, c)
		}
	}
	m.characters = chars
	return m
}


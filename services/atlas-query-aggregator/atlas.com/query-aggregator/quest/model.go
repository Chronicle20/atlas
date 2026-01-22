package quest

import (
	"strconv"
	"time"
)

// State represents the state of a quest
type State byte

const (
	StateNotStarted State = 0
	StateStarted    State = 1
	StateCompleted  State = 2
)

// Aliases for backward compatibility with tests
const (
	NOT_STARTED = StateNotStarted
	STARTED     = StateStarted
	COMPLETED   = StateCompleted
	UNDEFINED   = State(255) // Used when quest is not found
)

// QuestStatus is an alias for State for backward compatibility
type QuestStatus = State

// String returns the string representation of the quest state
func (s State) String() string {
	switch s {
	case StateNotStarted:
		return "NOT_STARTED"
	case StateStarted:
		return "STARTED"
	case StateCompleted:
		return "COMPLETED"
	default:
		return "NOT_STARTED"
	}
}

// StateFromByte creates a State from a byte value
func StateFromByte(b byte) State {
	switch b {
	case 0:
		return StateNotStarted
	case 1:
		return StateStarted
	case 2:
		return StateCompleted
	default:
		return StateNotStarted
	}
}

// ProgressModel represents quest progress for a specific info number
type ProgressModel struct {
	infoNumber uint32
	progress   string
}

func (m ProgressModel) InfoNumber() uint32 {
	return m.infoNumber
}

func (m ProgressModel) Progress() string {
	return m.progress
}

func (m ProgressModel) ProgressInt() int {
	val, err := strconv.Atoi(m.progress)
	if err != nil {
		return 0
	}
	return val
}

// Model represents a quest and its progress
type Model struct {
	characterId    uint32
	questId        uint32
	state          State
	startedAt      time.Time
	completedAt    time.Time
	progress       []ProgressModel
	progressByKey  map[string]int // For string-based progress lookup (used by tests)
}

// NewModel creates a new quest model
func NewModel(characterId uint32, questId uint32, state State) Model {
	return Model{
		characterId:   characterId,
		questId:       questId,
		state:         state,
		progress:      make([]ProgressModel, 0),
		progressByKey: make(map[string]int),
	}
}

// CharacterId returns the character ID
func (m Model) CharacterId() uint32 {
	return m.characterId
}

// QuestId returns the quest ID
func (m Model) QuestId() uint32 {
	return m.questId
}

// Id returns the quest ID (alias for QuestId)
func (m Model) Id() uint32 {
	return m.questId
}

// State returns the quest state
func (m Model) State() State {
	return m.state
}

// Status returns the quest state (alias for State)
func (m Model) Status() State {
	return m.state
}

// StartedAt returns when the quest was started
func (m Model) StartedAt() time.Time {
	return m.startedAt
}

// CompletedAt returns when the quest was completed
func (m Model) CompletedAt() time.Time {
	return m.completedAt
}

// Progress returns all progress entries
func (m Model) Progress() []ProgressModel {
	return m.progress
}

// GetProgress returns the progress for a specific info number
func (m Model) GetProgress(infoNumber uint32) (ProgressModel, bool) {
	for _, p := range m.progress {
		if p.infoNumber == infoNumber {
			return p, true
		}
	}
	return ProgressModel{}, false
}

// GetProgressByKey returns the progress value for a specific key (info number as string)
func (m Model) GetProgressByKey(key string) int {
	// First check the string-keyed map (used by builder)
	if m.progressByKey != nil {
		if val, found := m.progressByKey[key]; found {
			return val
		}
	}
	// Fall back to numeric lookup
	infoNumber, err := strconv.ParseUint(key, 10, 32)
	if err != nil {
		return 0
	}
	if p, found := m.GetProgress(uint32(infoNumber)); found {
		return p.ProgressInt()
	}
	return 0
}

// ProgressRestModel represents the REST representation of quest progress
type ProgressRestModel struct {
	InfoNumber uint32 `json:"infoNumber"`
	Progress   string `json:"progress"`
}

// RestModel represents the REST representation of a quest
type RestModel struct {
	Id          uint32              `json:"-"`
	CharacterId uint32              `json:"characterId"`
	QuestId     uint32              `json:"questId"`
	State       State               `json:"state"`
	StartedAt   time.Time           `json:"startedAt"`
	CompletedAt time.Time           `json:"completedAt,omitempty"`
	Progress    []ProgressRestModel `json:"progress"`
}

func (r RestModel) GetName() string {
	return "quest-status"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// Extract transforms a RestModel into a domain Model
func Extract(r RestModel) (Model, error) {
	progress := make([]ProgressModel, 0, len(r.Progress))
	for _, p := range r.Progress {
		progress = append(progress, ProgressModel{
			infoNumber: p.InfoNumber,
			progress:   p.Progress,
		})
	}

	return Model{
		characterId: r.CharacterId,
		questId:     r.QuestId,
		state:       r.State,
		startedAt:   r.StartedAt,
		completedAt: r.CompletedAt,
		progress:    progress,
	}, nil
}

// ModelBuilder provides a fluent API for building quest models
type ModelBuilder struct {
	characterId uint32
	questId     uint32
	state       State
	startedAt   time.Time
	completedAt time.Time
	progress    map[string]int
}

// NewModelBuilder creates a new ModelBuilder
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{
		progress: make(map[string]int),
	}
}

// SetCharacterId sets the character ID
func (b *ModelBuilder) SetCharacterId(id uint32) *ModelBuilder {
	b.characterId = id
	return b
}

// SetId sets the quest ID
func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.questId = id
	return b
}

// SetQuestId sets the quest ID (alias for SetId)
func (b *ModelBuilder) SetQuestId(id uint32) *ModelBuilder {
	b.questId = id
	return b
}

// SetStatus sets the quest state
func (b *ModelBuilder) SetStatus(state State) *ModelBuilder {
	b.state = state
	return b
}

// SetState sets the quest state (alias for SetStatus)
func (b *ModelBuilder) SetState(state State) *ModelBuilder {
	b.state = state
	return b
}

// SetStartedAt sets the started time
func (b *ModelBuilder) SetStartedAt(t time.Time) *ModelBuilder {
	b.startedAt = t
	return b
}

// SetCompletedAt sets the completed time
func (b *ModelBuilder) SetCompletedAt(t time.Time) *ModelBuilder {
	b.completedAt = t
	return b
}

// SetProgress sets a progress entry by key (info number as string)
func (b *ModelBuilder) SetProgress(key string, value int) *ModelBuilder {
	b.progress[key] = value
	return b
}

// Build creates the Model from the builder
func (b *ModelBuilder) Build() Model {
	progressModels := make([]ProgressModel, 0, len(b.progress))
	progressByKey := make(map[string]int, len(b.progress))

	for key, value := range b.progress {
		// Store in the string-keyed map for direct lookup
		progressByKey[key] = value

		// Also try to store as numeric info number for backward compatibility
		infoNumber, err := strconv.ParseUint(key, 10, 32)
		if err != nil {
			// If key is not a number, use 0
			infoNumber = 0
		}
		progressModels = append(progressModels, ProgressModel{
			infoNumber: uint32(infoNumber),
			progress:   strconv.Itoa(value),
		})
	}

	return Model{
		characterId:   b.characterId,
		questId:       b.questId,
		state:         b.state,
		startedAt:     b.startedAt,
		completedAt:   b.completedAt,
		progress:      progressModels,
		progressByKey: progressByKey,
	}
}
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
	characterId uint32
	questId     uint32
	state       State
	startedAt   time.Time
	completedAt time.Time
	progress    []ProgressModel
}

// NewModel creates a new quest model
func NewModel(characterId uint32, questId uint32, state State) Model {
	return Model{
		characterId: characterId,
		questId:     questId,
		state:       state,
		progress:    make([]ProgressModel, 0),
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

// State returns the quest state
func (m Model) State() State {
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
	CharacterId uint32              `json:"characterId"`
	QuestId     uint32              `json:"questId"`
	State       State               `json:"state"`
	StartedAt   time.Time           `json:"startedAt"`
	CompletedAt time.Time           `json:"completedAt,omitempty"`
	Progress    []ProgressRestModel `json:"progress"`
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
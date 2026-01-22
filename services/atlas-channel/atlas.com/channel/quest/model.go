package quest

import "time"

type State byte

const (
	StateNotStarted State = 0
	StateStarted    State = 1
	StateCompleted  State = 2
)

type Progress struct {
	infoNumber uint32
	progress   string
}

func (p Progress) InfoNumber() uint32 {
	return p.infoNumber
}

func (p Progress) Progress() string {
	return p.progress
}

type Model struct {
	id             uint32
	characterId    uint32
	questId        uint32
	state          State
	startedAt      time.Time
	completedAt    time.Time
	expirationTime time.Time
	completedCount uint32
	forfeitCount   uint32
	progress       []Progress
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

func (m Model) Progress() []Progress {
	return m.progress
}

func (m Model) GetProgress(infoNumber uint32) (Progress, bool) {
	for _, p := range m.progress {
		if p.InfoNumber() == infoNumber {
			return p, true
		}
	}
	return Progress{}, false
}

func (m Model) ProgressString() string {
	result := ""
	for _, p := range m.progress {
		result += p.Progress()
	}
	return result
}

// Helper functions for filtering quests by state

func FilterByState(quests []Model, state State) []Model {
	result := make([]Model, 0)
	for _, q := range quests {
		if q.State() == state {
			result = append(result, q)
		}
	}
	return result
}

func Started(quests []Model) []Model {
	return FilterByState(quests, StateStarted)
}

func Completed(quests []Model) []Model {
	return FilterByState(quests, StateCompleted)
}

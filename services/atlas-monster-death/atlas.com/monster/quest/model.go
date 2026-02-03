package quest

type State byte

const (
	StateNotStarted State = 0
	StateStarted    State = 1
	StateCompleted  State = 2
)

type Model struct {
	characterId uint32
	questId     uint32
	state       State
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

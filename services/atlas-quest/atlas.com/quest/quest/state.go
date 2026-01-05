package quest

type State byte

const (
	StateNotStarted State = 0
	StateStarted    State = 1
	StateCompleted  State = 2
)

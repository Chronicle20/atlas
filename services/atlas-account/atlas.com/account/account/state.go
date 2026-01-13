package account

type State uint8

const (
	StateNotLoggedIn State = 0
	StateLoggedIn    State = 1
	StateTransition  State = 2
)

func IsLoggedIn(s State) bool {
	return s != StateNotLoggedIn
}

func IsTransition(s State) bool {
	return s == StateTransition
}

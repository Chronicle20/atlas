package marriage

// MarriageState represents the lifecycle state of a marriage
type MarriageState uint8

const (
	// StateProposed represents a marriage proposal that has been sent but not yet accepted
	StateProposed MarriageState = iota
	// StateEngaged represents a marriage where the proposal has been accepted
	StateEngaged
	// StateMarried represents an active marriage after ceremony completion
	StateMarried
	// StateDivorced represents a terminated marriage through divorce
	StateDivorced
	// StateExpired represents a proposal that has expired without acceptance
	StateExpired
)

// String returns the string representation of MarriageState
func (s MarriageState) String() string {
	switch s {
	case StateProposed:
		return "proposed"
	case StateEngaged:
		return "engaged"
	case StateMarried:
		return "married"
	case StateDivorced:
		return "divorced"
	case StateExpired:
		return "expired"
	default:
		return "unknown"
	}
}

// IsActive returns true if the marriage state represents an active relationship
func (s MarriageState) IsActive() bool {
	return s == StateMarried
}

// IsTerminated returns true if the marriage state represents a terminated relationship
func (s MarriageState) IsTerminated() bool {
	return s == StateDivorced || s == StateExpired
}

// CanTransitionTo returns true if the marriage can transition to the target state
func (s MarriageState) CanTransitionTo(target MarriageState) bool {
	switch s {
	case StateProposed:
		return target == StateEngaged || target == StateExpired
	case StateEngaged:
		return target == StateMarried
	case StateMarried:
		return target == StateDivorced
	case StateDivorced, StateExpired:
		return false // Terminal states
	default:
		return false
	}
}

// ValidTransitions returns all valid transition states from the current state
func (s MarriageState) ValidTransitions() []MarriageState {
	switch s {
	case StateProposed:
		return []MarriageState{StateEngaged, StateExpired}
	case StateEngaged:
		return []MarriageState{StateMarried}
	case StateMarried:
		return []MarriageState{StateDivorced}
	case StateDivorced, StateExpired:
		return []MarriageState{} // Terminal states
	default:
		return []MarriageState{}
	}
}
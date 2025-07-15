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

// CeremonyState represents the lifecycle state of a ceremony
type CeremonyState uint8

const (
	// CeremonyStateScheduled represents a ceremony that has been scheduled but not yet started
	CeremonyStateScheduled CeremonyState = iota
	// CeremonyStateActive represents a ceremony that is currently in progress
	CeremonyStateActive
	// CeremonyStateCompleted represents a ceremony that has been successfully completed
	CeremonyStateCompleted
	// CeremonyStateCancelled represents a ceremony that has been cancelled
	CeremonyStateCancelled
	// CeremonyStatePostponed represents a ceremony that has been postponed due to disconnection
	CeremonyStatePostponed
)

// String returns the string representation of CeremonyState
func (s CeremonyState) String() string {
	switch s {
	case CeremonyStateScheduled:
		return "scheduled"
	case CeremonyStateActive:
		return "active"
	case CeremonyStateCompleted:
		return "completed"
	case CeremonyStateCancelled:
		return "cancelled"
	case CeremonyStatePostponed:
		return "postponed"
	default:
		return "unknown"
	}
}

// IsActive returns true if the ceremony state represents an active ceremony
func (s CeremonyState) IsActive() bool {
	return s == CeremonyStateActive
}

// IsTerminated returns true if the ceremony state represents a terminated ceremony
func (s CeremonyState) IsTerminated() bool {
	return s == CeremonyStateCompleted || s == CeremonyStateCancelled
}

// IsInProgress returns true if the ceremony is either scheduled or active
func (s CeremonyState) IsInProgress() bool {
	return s == CeremonyStateScheduled || s == CeremonyStateActive
}

// CanTransitionTo returns true if the ceremony can transition to the target state
func (s CeremonyState) CanTransitionTo(target CeremonyState) bool {
	switch s {
	case CeremonyStateScheduled:
		return target == CeremonyStateActive || target == CeremonyStateCancelled
	case CeremonyStateActive:
		return target == CeremonyStateCompleted || target == CeremonyStateCancelled || target == CeremonyStatePostponed
	case CeremonyStatePostponed:
		return target == CeremonyStateScheduled || target == CeremonyStateActive || target == CeremonyStateCancelled
	case CeremonyStateCompleted, CeremonyStateCancelled:
		return false // Terminal states
	default:
		return false
	}
}

// ValidTransitions returns all valid transition states from the current state
func (s CeremonyState) ValidTransitions() []CeremonyState {
	switch s {
	case CeremonyStateScheduled:
		return []CeremonyState{CeremonyStateActive, CeremonyStateCancelled}
	case CeremonyStateActive:
		return []CeremonyState{CeremonyStateCompleted, CeremonyStateCancelled, CeremonyStatePostponed}
	case CeremonyStatePostponed:
		return []CeremonyState{CeremonyStateScheduled, CeremonyStateActive, CeremonyStateCancelled}
	case CeremonyStateCompleted, CeremonyStateCancelled:
		return []CeremonyState{} // Terminal states
	default:
		return []CeremonyState{}
	}
}
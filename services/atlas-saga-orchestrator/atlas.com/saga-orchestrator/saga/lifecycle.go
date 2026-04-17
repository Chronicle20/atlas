package saga

// SagaLifecycleState tracks the terminal-state machine of a saga cache entry.
// It is distinct from per-step Status: a saga's lifecycle can be Compensating
// even though individual steps are Completed/Failed.
//
// Transitions (enforced by Cache.TryTransition):
//
//	Pending → Compensating → Failed
//	Pending → Completed
//
// The terminal-state guard ensures exactly one Failed emission per non-completing
// saga under timer / StepCompleted races. See PRD §4.7 / plan Phase 2.
type SagaLifecycleState string

const (
	SagaLifecyclePending      SagaLifecycleState = "pending"
	SagaLifecycleCompensating SagaLifecycleState = "compensating"
	SagaLifecycleFailed       SagaLifecycleState = "failed"
	SagaLifecycleCompleted    SagaLifecycleState = "completed"
)

// IsValidTransition reports whether `from → to` is a permitted lifecycle transition.
// Self-transitions are rejected. Terminal states have no outgoing edges.
func IsValidTransition(from, to SagaLifecycleState) bool {
	switch from {
	case SagaLifecyclePending:
		return to == SagaLifecycleCompensating || to == SagaLifecycleCompleted
	case SagaLifecycleCompensating:
		return to == SagaLifecycleFailed
	default:
		return false
	}
}

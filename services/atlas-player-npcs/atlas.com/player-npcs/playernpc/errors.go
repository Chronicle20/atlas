package playernpc

import "errors"

// The four design §8.3 failure codes. REST (Task 16) maps each to 409 with
// a `code` field carrying the string; Kafka command failures (Task 17) and
// the GM command (design §9.2) branch on the same codes. `errors.Is` finds
// these directly since Deploy/Redeploy/Remove return them (or a wrapped
// allocation.ErrPoolExhausted/position.ErrMapFull, which carry the same
// text) verbatim, never behind a generic error.
const (
	CodePoolExhausted = "pool_exhausted"
	CodeMapFull       = "map_full"
	CodeDuplicate     = "duplicate"
	CodeIneligible    = "ineligible"
)

var (
	// ErrPoolExhausted is returned when no usable script id remains in the
	// branch or the pool (allocation.ErrPoolExhausted, PRD §5).
	ErrPoolExhausted = errors.New(CodePoolExhausted)
	// ErrMapFull is returned when no free position exists at the maximum
	// step (position.ErrMapFull, PRD §5).
	ErrMapFull = errors.New(CodeMapFull)
	// ErrDuplicate is returned when the character already has a Player NPC
	// deployed on this map (PRD §5).
	ErrDuplicate = errors.New(CodeDuplicate)
	// ErrIneligible is returned when the eligibility-checked path's
	// level/GM checks fail (eligibility.ReasonIneligible, PRD §5).
	ErrIneligible = errors.New(CodeIneligible)
)

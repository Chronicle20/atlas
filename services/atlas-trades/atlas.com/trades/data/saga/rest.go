// Package saga is the atlas-saga-orchestrator REST client. It answers one
// question, asked only by startup reconciliation: what happened to the
// settlement saga with this transaction id?
//
// The orchestrator's saga store is Postgres-backed and its terminal states are
// DURABLE, which is what makes the question answerable at all after a restart:
//
//   - completion is a SOFT delete — "Remove marks a saga as completed (soft
//     delete)" sets status='completed' rather than deleting the row
//     (saga/store.go:203-212);
//   - failure sets status='failed' (store.go:252-258), and both terminal states
//     are preserved against any later write: "Terminal-preserving: failed/
//     completed can never be overwritten by a recomputed active/compensating
//     status" (store.go:127-131);
//   - GetById reads by transaction id with NO status filter (store.go:73-76), so
//     a terminal saga is still returned;
//   - nothing in the service deletes saga rows, so the record does not age out.
//
// The REST resource (saga/resource.go:22) exposes GET /sagas/{transactionId}.
// Its RestModel carries no saga-level status field — only per-step Status — so
// the outcome is derived from the steps, which is what Saga.Failing() itself
// does (any failed step ⇒ the saga is failing).
package saga

import (
	"github.com/google/uuid"
)

// Outcome is what reconciliation concluded about a submitted saga.
type Outcome string

const (
	// OutcomeSucceeded means every step completed: the swap executed.
	OutcomeSucceeded Outcome = "SUCCEEDED"
	// OutcomeFailed means at least one step failed, so the saga compensated.
	OutcomeFailed Outcome = "FAILED"
	// OutcomeRunning means the saga still has pending steps. It is not a
	// terminal answer and must never be treated as either of the others.
	OutcomeRunning Outcome = "RUNNING"
)

// Step statuses as the orchestrator serialises them (libs/atlas-saga's Status).
const (
	stepPending   = "pending"
	stepCompleted = "completed"
	stepFailed    = "failed"
)

// StepRestModel mirrors the fields of the orchestrator's StepRestModel that
// this reader needs. `payload` is deliberately unmapped — an unmapped json
// field is discarded by the decoder, and reconciliation reads its item and meso
// figures from its OWN durable record, never from the orchestrator's copy.
type StepRestModel struct {
	StepId string `json:"stepId"`
	Status string `json:"status"`
	Action string `json:"action"`
}

// RestModel mirrors atlas-saga-orchestrator's saga resource.
type RestModel struct {
	TransactionId uuid.UUID       `json:"-"`
	SagaType      string          `json:"sagaType"`
	InitiatedBy   string          `json:"initiatedBy"`
	Steps         []StepRestModel `json:"steps"`
}

func (r RestModel) GetName() string { return "sagas" }

func (r RestModel) GetID() string { return r.TransactionId.String() }

func (r *RestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.TransactionId = id
	return nil
}

func (r *RestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Extract derives the saga's outcome from its steps.
//
// A saga with NO steps is reported as RUNNING, not as succeeded: an empty step
// list means the composite has not been expanded yet, and reading "no failures"
// as success there would complete a trade that has not moved anything.
func Extract(rm RestModel) (Outcome, error) {
	if len(rm.Steps) == 0 {
		return OutcomeRunning, nil
	}
	pending := false
	for _, s := range rm.Steps {
		switch s.Status {
		case stepFailed:
			// One failed step is decisive: the orchestrator compensates the
			// whole saga from it (Saga.Failing()).
			return OutcomeFailed, nil
		case stepPending:
			pending = true
		case stepCompleted:
		default:
			// An unrecognised status is not evidence of anything. Treating it
			// as terminal would either credit or cancel a trade on a guess.
			pending = true
		}
	}
	if pending {
		return OutcomeRunning, nil
	}
	return OutcomeSucceeded, nil
}

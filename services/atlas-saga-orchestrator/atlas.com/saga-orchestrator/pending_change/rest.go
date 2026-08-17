package pending_change

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// WorldChangeInputRestModel is the POST body of atlas-character's dedicated
// world-change route (Task 7): {data:{type:"characters",attributes:{newWorldId,transactionId}}}.
// It mirrors character/rest.go's WorldChangeInputRestModel in atlas-character
// exactly — a PATCH through the generic characters resource cannot express
// "transfer to world 0" because world.Id is a byte and PATCH treats a zero
// field as absent (task-227 controller ruling).
type WorldChangeInputRestModel struct {
	Id            string    `json:"-"`
	NewWorldId    world.Id  `json:"newWorldId"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}

func (r WorldChangeInputRestModel) GetName() string {
	return "characters"
}

func (r WorldChangeInputRestModel) GetID() string {
	return r.Id
}

func (r *WorldChangeInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// ResolveInputRestModel is the POST body of atlas-character's generic
// pending-change resolve route: {data:{type:"pending-changes",attributes:{status,reason}}}.
// Mirrors pending_change/rest.go's ResolveInputRestModel in atlas-character.
type ResolveInputRestModel struct {
	Id     string `json:"-"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (r ResolveInputRestModel) GetName() string {
	return "pending-changes"
}

func (r ResolveInputRestModel) GetID() string {
	return r.Id
}

func (r *ResolveInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

const (
	StatusApplied  = "APPLIED"
	StatusRejected = "REJECTED"
)

// EligibilityRestModel is the read-only response of GET
// .../transfer-eligibility. Mirrors pending_change/rest.go's
// EligibilityRestModel in atlas-character exactly.
type EligibilityRestModel struct {
	Id       string `json:"-"`
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

func (r EligibilityRestModel) GetName() string {
	return "transfer-eligibilities"
}

func (r EligibilityRestModel) GetID() string {
	return r.Id
}

func (r *EligibilityRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

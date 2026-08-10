package coupon

import (
	"atlas-cashshop/coupon/reward"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Nullable distinguishes the three states a JSON field can be in — ABSENT,
// explicitly null, and present with a value — which a plain pointer cannot:
// encoding/json sets a pointer to nil for BOTH an absent field and a literal
// null, at any depth (**T included).
//
// Set is written only by UnmarshalJSON, which encoding/json calls for a
// literal null as well as for a value, so Set == false means "the key was not
// in the document at all".
type Nullable[T any] struct {
	Set   bool
	Value *T
}

func (n *Nullable[T]) UnmarshalJSON(b []byte) error {
	n.Set = true
	if string(b) == "null" {
		n.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}

// Or resolves the field against the stored value: absent preserves it,
// present (including an explicit null) replaces it.
func (n Nullable[T]) Or(current *T) *T {
	if !n.Set {
		return current
	}
	return n.Value
}

// PatchRestModel is the PATCH /coupons/{id} body. It is a SEPARATE type from
// RestModel because PATCH semantics are genuinely partial and RestModel cannot
// express absence: its `Active bool` would read an omitted field as false —
// deactivating a coupon nobody asked to deactivate — and its `*time.Time`
// fields would read an omitted expiresAt as "clear the expiry", turning a
// coupon that expired last January into a never-expiring one. That is silent
// data loss on an admin surface, so the two shapes are kept apart rather than
// one being reused for both verbs.
//
// FIELD SEMANTICS
//
//	description  absent -> preserve; a string -> set ("" clears it — a
//	             description has a natural empty value, so it is not nullable)
//	active       absent -> preserve; true/false -> set
//	startsAt     absent -> preserve; null -> clear; a timestamp -> set
//	expiresAt    absent -> preserve; null -> clear; a timestamp -> set
//	maxUses      absent -> preserve; null -> clear (unlimited); a number -> set
//	rewards      absent -> preserve; a bundle -> replace. An explicit null is
//	             NOT a way to empty the bundle: it reaches Rewards.Validate,
//	             which refuses, because a coupon must grant at least one reward.
//
// NOT EDITABLE, and ignored wherever they appear in the body: code (the
// identity a player types), batchId (the generation that produced the row),
// redemptionCount (owned by reserveUse) and id (taken from the path).
type PatchRestModel struct {
	Id          uuid.UUID
	Description Nullable[string]         `json:"description"`
	Active      Nullable[bool]           `json:"active"`
	StartsAt    Nullable[time.Time]      `json:"startsAt"`
	ExpiresAt   Nullable[time.Time]      `json:"expiresAt"`
	MaxUses     Nullable[uint32]         `json:"maxUses"`
	Rewards     Nullable[reward.Rewards] `json:"rewards"`
}

// GetName makes api2go's checkType accept a `"type": "coupons"` body for this
// input shape — a PATCH names the resource it edits, not the shape of the
// patch.
func (r PatchRestModel) GetName() string {
	return "coupons"
}

func (r PatchRestModel) GetID() string {
	return r.Id.String()
}

func (r *PatchRestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = uuid.Nil
		return nil
	}
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

// Apply merges the patch onto the stored coupon and re-runs every Builder
// invariant over the RESULT, so a field the admin did not mention still has to
// be consistent with the ones they did.
//
// redemptionCount comes from the stored row, which is what makes "maxUses
// below the current redemption count" a rejection rather than a silent shrink.
func (r PatchRestModel) Apply(current Model) (Model, error) {
	description := current.Description()
	if r.Description.Set {
		// An explicit null and "" both clear it; there is no third state for
		// a plain string field.
		description = ""
		if r.Description.Value != nil {
			description = *r.Description.Value
		}
	}

	// active is not nullable — a coupon is either redeemable or it is not —
	// so an explicit null preserves the stored value rather than inventing
	// a third state.
	active := current.Active()
	if r.Active.Set && r.Active.Value != nil {
		active = *r.Active.Value
	}

	rewards := current.Rewards()
	if r.Rewards.Set {
		if r.Rewards.Value == nil {
			rewards = nil
		} else {
			rewards = *r.Rewards.Value
		}
	}

	return NewBuilder(current.Code()).
		SetId(current.Id()).
		SetBatchId(current.BatchId()).
		SetDescription(description).
		SetActive(active).
		SetStartsAt(r.StartsAt.Or(current.StartsAt())).
		SetExpiresAt(r.ExpiresAt.Or(current.ExpiresAt())).
		SetMaxUses(r.MaxUses.Or(current.MaxUses())).
		SetRedemptionCount(current.RedemptionCount()).
		SetRewards(rewards).
		Build()
}

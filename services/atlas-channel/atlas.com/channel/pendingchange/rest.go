package pendingchange

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the read-side JSON:API projection of a pending change record,
// as produced by atlas-character (services/atlas-character/atlas.com/character/pending_change/rest.go).
type RestModel struct {
	Id                 string     `json:"-"`
	CharacterId        uint32     `json:"characterId"`
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	RequestedName      string     `json:"requestedName,omitempty"`
	DestinationWorldId world.Id   `json:"destinationWorldId"`
	SourceWorldId      world.Id   `json:"sourceWorldId"`
	Reason             string     `json:"reason,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	ExpiresAt          time.Time  `json:"expiresAt"`
	ResolvedAt         *time.Time `json:"resolvedAt,omitempty"`
}

func (r RestModel) GetName() string {
	return "pending-changes"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// CreateInputRestModel is the POST body atlas-character's
// POST /characters/{characterId}/pending-changes expects:
// {data:{type:"pending-changes",attributes:{type,requestedName,destinationWorldId}}}.
//
// atlas-character's input model also accepts an assetId, for the item path
// where a coupon in the player's inventory is consumed at request acceptance.
// This service has no such path — its only producers are the cash-shop purchase
// handlers — so the field is deliberately absent here rather than present and
// always nil: on the purchase path an assetId is not merely unused, it is
// actively wrong (it drives a destroy_asset saga for a coupon the player does
// not hold, and a refund on cancel for a consumption that never happened).
type CreateInputRestModel struct {
	Id                 string   `json:"-"`
	Type               string   `json:"type"`
	RequestedName      string   `json:"requestedName,omitempty"`
	DestinationWorldId world.Id `json:"destinationWorldId,omitempty"`
}

func (r CreateInputRestModel) GetName() string {
	return "pending-changes"
}

func (r CreateInputRestModel) GetID() string {
	return r.Id
}

func (r *CreateInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// CancelInputRestModel is the POST body atlas-character's self-scoped
// POST /characters/{characterId}/pending-changes/cancel expects:
// {data:{type:"pending-changes",attributes:{type}}}. It carries no record
// id -- the wire packet that drives this call has none (task-227
// client-cancel addendum); atlas-character resolves the record by
// (characterId, type) instead.
type CancelInputRestModel struct {
	Id   string `json:"-"`
	Type string `json:"type"`
}

func (r CancelInputRestModel) GetName() string {
	return "pending-changes"
}

func (r CancelInputRestModel) GetID() string {
	return r.Id
}

func (r *CancelInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// EligibilityRestModel is the read-only response of atlas-character's
// GET .../transfer-eligibility-independent (services/atlas-character/atlas.com/character/pending_change/rest.go's
// EligibilityRestModel, which the destination-bearing
// .../transfer-eligibility route shares). Id is the characterId the check
// was run for, so the response is addressable JSON:API; only Eligible/Reason
// are read here.
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

package pending_change

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the read-side JSON:API projection of a pending change record.
// AssetId is deliberately absent: it is an internal detail of the refund
// mechanism (design §5.4), not something an operator-facing GET needs to
// expose.
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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                 m.Id().String(),
		CharacterId:        m.CharacterId(),
		Type:               m.Type(),
		Status:             m.Status(),
		RequestedName:      m.RequestedName(),
		DestinationWorldId: m.DestinationWorldId(),
		SourceWorldId:      m.SourceWorldId(),
		Reason:             m.Reason(),
		CreatedAt:          m.CreatedAt(),
		ExpiresAt:          m.ExpiresAt(),
		ResolvedAt:         m.ResolvedAt(),
	}, nil
}

// CreateInputRestModel is the POST body:
// {data:{type:"pending-changes",attributes:{type,requestedName,destinationWorldId,assetId}}}.
//
// AssetId carries an item TEMPLATE id, not an instance id: it travels
// verbatim onto sharedsaga.DestroyAssetPayload.TemplateId at every call site
// in this package's producer (Task 6's award/destroy providers), and every
// other caller of that saga contract also treats the field as a template id.
// atlas-channel (Task 24) is the other side of this seam and must agree on
// that semantics.
type CreateInputRestModel struct {
	Id                 string   `json:"-"`
	Type               string   `json:"type"`
	RequestedName      string   `json:"requestedName,omitempty"`
	DestinationWorldId world.Id `json:"destinationWorldId,omitempty"`
	AssetId            *uint32  `json:"assetId,omitempty"`
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

// EligibilityRestModel is the read-only response of GET
// .../transfer-eligibility. Id is the characterId the check was run for, so
// the response is addressable JSON:API, but the caller (atlas-channel) only
// ever reads Eligible/Reason.
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

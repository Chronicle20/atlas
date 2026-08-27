package ring

import (
	"fmt"

	"github.com/google/uuid"
)

// RestModel is the channel-side JSON:API decode target for atlas-cashshop's
// GET /rings route (task-269 task 8). Read-only: the producing RestModel
// (atlas-cashshop/ring/rest.go) has only a Transform, no Extract, so this
// package writes its own decode side rather than reusing a symmetric
// function. Transform is nonetheless defined here (DOM-04) for callers that
// need to re-encode a cached Model back to the wire shape.
type RestModel struct {
	Id                 string `json:"-"`
	PairId             string `json:"pairId"`
	CharacterId        uint32 `json:"characterId"`
	PartnerCharacterId uint32 `json:"partnerCharacterId"`
	AssetId            uint32 `json:"assetId"`
	ItemTemplateId     uint32 `json:"itemTemplateId"`
	RingType           string `json:"ringType"`
	State              string `json:"state"`
	CashId             int64  `json:"cashId"`
	PartnerCashId      int64  `json:"partnerCashId"`
	PartnerName        string `json:"partnerName"`
}

func (r RestModel) GetName() string {
	return "rings"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID satisfies the api2go UnmarshalToOneRelations interface.
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

// SetToManyReferenceIDs satisfies the api2go UnmarshalToManyRelations interface.
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Transform converts a ring.Model back to its RestModel wire shape.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                 m.Id().String(),
		PairId:             m.PairId().String(),
		CharacterId:        m.CharacterId(),
		PartnerCharacterId: m.PartnerCharacterId(),
		ItemTemplateId:     m.ItemTemplateId(),
		RingType:           string(m.Type()),
		State:              string(m.State()),
		CashId:             m.CashId(),
		PartnerCashId:      m.PartnerCashId(),
		PartnerName:        m.PartnerName(),
	}, nil
}

// Extract converts the wire RestModel into a Model, rejecting anything the
// wire representation cannot faithfully carry: an unparseable id/pairId, or
// an unrecognised RingType/State. Defaulting either silently would fabricate
// a pairing (or its outcome) that the producing side never recorded.
func Extract(rm RestModel) (Model, error) {
	id, err := uuid.Parse(rm.Id)
	if err != nil {
		return Model{}, err
	}
	pairId, err := uuid.Parse(rm.PairId)
	if err != nil {
		return Model{}, err
	}
	t, err := parseType(rm.RingType)
	if err != nil {
		return Model{}, err
	}
	s, err := parseState(rm.State)
	if err != nil {
		return Model{}, err
	}
	return NewModelBuilder(id, pairId).
		SetCharacterId(rm.CharacterId).
		SetPartnerCharacterId(rm.PartnerCharacterId).
		SetItemTemplateId(rm.ItemTemplateId).
		SetType(t).
		SetState(s).
		SetCashId(rm.CashId).
		SetPartnerCashId(rm.PartnerCashId).
		SetPartnerName(rm.PartnerName).
		Build()
}

func parseType(s string) (Type, error) {
	switch Type(s) {
	case TypeCouple, TypeFriendship:
		return Type(s), nil
	default:
		return "", fmt.Errorf("unknown ring type %q", s)
	}
}

func parseState(s string) (State, error) {
	switch State(s) {
	case StateActive, StateBroken, StateExpired:
		return State(s), nil
	default:
		return "", fmt.Errorf("unknown ring state %q", s)
	}
}

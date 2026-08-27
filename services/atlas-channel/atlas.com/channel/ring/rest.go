package ring

import (
	"fmt"

	"github.com/google/uuid"
)

// RestModel is the channel-side JSON:API decode target for atlas-cashshop's
// GET /rings route (task-269 task 8). Read-only: the producing RestModel
// (atlas-cashshop/ring/rest.go) has only a Transform, no Extract, so this
// package writes its own decode side rather than reusing a symmetric
// function.
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
	return Model{
		id:                 id,
		pairId:             pairId,
		characterId:        rm.CharacterId,
		partnerCharacterId: rm.PartnerCharacterId,
		itemTemplateId:     rm.ItemTemplateId,
		ringType:           t,
		state:              s,
		cashId:             rm.CashId,
		partnerCashId:      rm.PartnerCashId,
		partnerName:        rm.PartnerName,
	}, nil
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

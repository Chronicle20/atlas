package asset

import (
	"atlas-channel/cashshop/item"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel represents a cash shop inventory asset for REST API
type RestModel struct {
	Id            uint32    `json:"-"`
	CompartmentId uuid.UUID `json:"compartmentId"`
	CashId        int64     `json:"cashId,string"`
	TemplateId    uint32    `json:"templateId"`
	CommodityId   uint32    `json:"commodityId"`
	Quantity      uint32    `json:"quantity"`
	Flag          uint16    `json:"flag"`
	PurchasedBy   uint32    `json:"purchasedBy"`
	Expiration    time.Time `json:"expiration"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "assets"
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

// SetID sets the resource ID
func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		return nil
	}
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// GetReferences returns the references for this resource
func (r RestModel) GetReferences() []jsonapi.Reference {
	return nil
}

// GetReferencedIDs returns the referenced IDs for this resource
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return nil
}

// GetReferencedStructs returns the referenced structs for this resource
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return nil
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// SetReferencedStructs sets the referenced structs
func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// Transform converts an asset.Model to a RestModel
func Transform(a Model) (RestModel, error) {
	return RestModel{
		Id:            a.Id(),
		CompartmentId: a.CompartmentId(),
		CashId:        a.Item().CashId(),
		TemplateId:    a.Item().TemplateId(),
		CommodityId:   a.Item().CommodityId(),
		Quantity:      a.Item().Quantity(),
		Flag:          a.Item().Flag(),
		PurchasedBy:   a.Item().PurchasedBy(),
		Expiration:    a.Item().Expiration(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	i, err := item.Extract(item.RestModel{
		CashId:      rm.CashId,
		TemplateId:  rm.TemplateId,
		CommodityId: rm.CommodityId,
		Quantity:    rm.Quantity,
		Flag:        rm.Flag,
		PurchasedBy: rm.PurchasedBy,
		Expiration:  rm.Expiration,
	})
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:            rm.Id,
		compartmentId: rm.CompartmentId,
		item:          i,
	}, nil
}

package cashshop

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// AssetRestModel represents a cash shop inventory asset (flattened)
type AssetRestModel struct {
	Id            uint32    `json:"-"`
	CompartmentId string    `json:"compartmentId"`
	CashId        int64     `json:"cashId,string"`
	TemplateId    uint32    `json:"templateId"`
	CommodityId   uint32    `json:"commodityId"`
	Quantity      uint32    `json:"quantity"`
	Flag          uint16    `json:"flag"`
	PurchasedBy   uint32    `json:"purchasedBy"`
	Expiration    time.Time `json:"expiration"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *AssetRestModel) SetID(strId string) error {
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

// CompartmentRestModel represents a cash shop inventory compartment
type CompartmentRestModel struct {
	Id        uuid.UUID        `json:"-"`
	AccountId uint32           `json:"accountId"`
	Type      byte             `json:"type"`
	Capacity  uint32           `json:"capacity"`
	Assets    []AssetRestModel `json:"-"`
}

func (r CompartmentRestModel) GetName() string {
	return "compartments"
}

func (r CompartmentRestModel) GetID() string {
	return r.Id.String()
}

func (r *CompartmentRestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func (r CompartmentRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "assets",
			Name: "assets",
		},
	}
}

func (r CompartmentRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Assets {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: v.GetName(),
			Name: v.GetName(),
		})
	}
	return result
}

func (r CompartmentRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Assets {
		result = append(result, r.Assets[key])
	}
	return result
}

func (r *CompartmentRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *CompartmentRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "assets" {
		for _, idStr := range IDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			r.Assets = append(r.Assets, AssetRestModel{Id: uint32(id)})
		}
	}
	return nil
}

func (r *CompartmentRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["assets"]; ok {
		assets := make([]AssetRestModel, 0)
		for _, ri := range r.Assets {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				assets = append(assets, wip)
			}
		}
		r.Assets = assets
	}
	return nil
}

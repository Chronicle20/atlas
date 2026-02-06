package cashshop

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// ItemRestModel represents a cash item from the cash shop service
type ItemRestModel struct {
	Id          uint32 `json:"-"`
	CashId      int64  `json:"cashId,string"`
	TemplateId  uint32 `json:"templateId"`
	Quantity    uint32 `json:"quantity"`
	Flag        uint16 `json:"flag"`
	PurchasedBy uint32 `json:"purchasedBy"`
}

func (r ItemRestModel) GetName() string {
	return "items"
}

func (r ItemRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *ItemRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// AssetRestModel represents a cash shop inventory asset
type AssetRestModel struct {
	Id            uuid.UUID     `json:"-"`
	CompartmentId uuid.UUID     `json:"compartmentId"`
	Item          ItemRestModel `json:"-"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return r.Id.String()
}

func (r *AssetRestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func (r AssetRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "items",
			Name: "item",
		},
	}
}

func (r AssetRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{
		{
			ID:   r.Item.GetID(),
			Type: r.Item.GetName(),
			Name: "item",
		},
	}
}

func (r AssetRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{r.Item}
}

func (r *AssetRestModel) SetToOneReferenceID(name, ID string) error {
	if name == "item" {
		var item ItemRestModel
		if err := item.SetID(ID); err != nil {
			return err
		}
		r.Item = item
	}
	return nil
}

func (r *AssetRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *AssetRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if r.Item.GetID() != "" {
		if refMap, ok := references["items"]; ok {
			if ref, ok := refMap[r.Item.GetID()]; ok {
				err := jsonapi.ProcessIncludeData(&r.Item, ref, references)
				if err != nil {
					return err
				}
			}
		}
	}
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
			id, err := uuid.Parse(idStr)
			if err != nil {
				return err
			}
			r.Assets = append(r.Assets, AssetRestModel{Id: id})
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

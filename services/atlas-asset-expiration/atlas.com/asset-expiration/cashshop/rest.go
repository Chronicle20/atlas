package cashshop

import (
	"strconv"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
)

type InventoryRestModel struct {
	Id           string                `json:"-"`
	AccountId    uint32                `json:"accountId"`
	Compartments []CompartmentRestModel `json:"-"`
}

func (r InventoryRestModel) GetName() string {
	return "cash-inventories"
}

func (r InventoryRestModel) GetID() string {
	return r.Id
}

func (r *InventoryRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r InventoryRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "compartments",
			Name: "compartments",
		},
	}
}

func (r InventoryRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Compartments {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: v.GetName(),
			Name: "compartments",
		})
	}
	return result
}

func (r InventoryRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Compartments {
		result = append(result, r.Compartments[key])
	}
	return result
}

func (r *InventoryRestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *InventoryRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "compartments" {
		for _, idStr := range IDs {
			r.Compartments = append(r.Compartments, CompartmentRestModel{Id: idStr})
		}
	}
	return nil
}

func (r *InventoryRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["compartments"]; ok {
		compartments := make([]CompartmentRestModel, 0)
		for _, ri := range r.Compartments {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				compartments = append(compartments, wip)
			}
		}
		r.Compartments = compartments
	}
	return nil
}

type CompartmentRestModel struct {
	Id        string           `json:"-"`
	AccountId uint32           `json:"accountId"`
	Type      uint8            `json:"type"`
	Capacity  uint32           `json:"capacity"`
	Assets    []AssetRestModel `json:"-"`
}

func (r CompartmentRestModel) GetName() string {
	return "compartments"
}

func (r CompartmentRestModel) GetID() string {
	return r.Id
}

func (r *CompartmentRestModel) SetID(id string) error {
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
			Name: "assets",
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

func (r *CompartmentRestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *CompartmentRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "assets" {
		for _, idStr := range IDs {
			r.Assets = append(r.Assets, AssetRestModel{Id: idStr})
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

type AssetRestModel struct {
	Id            string        `json:"-"`
	CompartmentId string        `json:"compartmentId"`
	Item          ItemRestModel `json:"-"`
}

func (r AssetRestModel) GetName() string {
	return "assets"
}

func (r AssetRestModel) GetID() string {
	return r.Id
}

func (r *AssetRestModel) SetID(id string) error {
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
		r.Item.Id = ID
	}
	return nil
}

func (r *AssetRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

func (r *AssetRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["items"]; ok {
		if ref, ok := refMap[r.Item.GetID()]; ok {
			err := jsonapi.ProcessIncludeData(&r.Item, ref, references)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type ItemRestModel struct {
	Id          string    `json:"-"`
	CashId      int64     `json:"cashId,string"`
	TemplateId  uint32    `json:"templateId"`
	CommodityId uint32    `json:"commodityId"`
	Quantity    uint32    `json:"quantity"`
	Flag        uint16    `json:"flag"`
	PurchasedBy uint32    `json:"purchasedBy"`
	Expiration  time.Time `json:"expiration"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (r ItemRestModel) GetName() string {
	return "items"
}

func (r ItemRestModel) GetID() string {
	return r.Id
}

func (r *ItemRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *ItemRestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *ItemRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

// GetItemId parses the ID string to uint32
func (r ItemRestModel) GetItemId() uint32 {
	id, _ := strconv.ParseUint(r.Id, 10, 32)
	return uint32(id)
}

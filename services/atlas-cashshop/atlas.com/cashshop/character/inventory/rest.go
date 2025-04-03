package inventory

import (
	"atlas-cashshop/character/inventory/equipable"
	"atlas-cashshop/character/inventory/item"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Equipable EquipableRestModel `json:"equipable"`
	Useable   ItemRestModel      `json:"useable"`
	Setup     ItemRestModel      `json:"setup"`
	Etc       ItemRestModel      `json:"etc"`
	Cash      ItemRestModel      `json:"cash"`
}

type EquipableRestModel struct {
	Type     string                `json:"-"`
	Capacity uint32                `json:"capacity"`
	Items    []equipable.RestModel `json:"items"`
}

func (r EquipableRestModel) GetName() string {
	return "inventories"
}

func (r EquipableRestModel) GetID() string {
	return r.Type
}

func (r EquipableRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "equipables",
			Name: "equipables",
		},
	}
}

func (r EquipableRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Items {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: "equipables",
			Name: "equipables",
		})
	}
	return result
}

func (r EquipableRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Items {
		result = append(result, r.Items[key])
	}

	return result
}

func (r *EquipableRestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *EquipableRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "equipables" {
		for _, idStr := range IDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			r.Items = append(r.Items, equipable.RestModel{Id: uint32(id)})
		}
	}
	return nil
}

func (r *EquipableRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["equipables"]; ok {
		items := make([]equipable.RestModel, 0)
		for _, ri := range r.Items {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				items = append(items, wip)
			}
		}
		r.Items = items
	}
	return nil
}

type ItemRestModel struct {
	Type     string           `json:"-"`
	Capacity uint32           `json:"capacity"`
	Items    []item.RestModel `json:"items"`
}

func (r ItemRestModel) GetName() string {
	return "inventories"
}

func (r ItemRestModel) GetID() string {
	return r.Type
}

func (r ItemRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "items",
			Name: "items",
		},
	}
}

func (r ItemRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Items {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: "items",
			Name: "items",
		})
	}
	return result
}

func (r ItemRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Items {
		result = append(result, r.Items[key])
	}

	return result
}

func (r *ItemRestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *ItemRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "items" {
		for _, idStr := range IDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return err
			}
			r.Items = append(r.Items, item.RestModel{Id: uint32(id)})
		}
	}
	return nil
}

func (r *ItemRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["items"]; ok {
		items := make([]item.RestModel, 0)
		for _, ri := range r.Items {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				items = append(items, wip)
			}
		}
		r.Items = items
	}
	return nil
}

func Extract(model RestModel) (Model, error) {
	e, err := ExtractEquipable(model.Equipable)
	if err != nil {
		return Model{}, err
	}

	return Model{
		equipable: e,
		useable:   ExtractItem(model.Useable),
		setup:     ExtractItem(model.Setup),
		etc:       ExtractItem(model.Etc),
		cash:      ExtractItem(model.Cash),
	}, nil
}

func ExtractItem(rm ItemRestModel) ItemModel {
	return ItemModel{
		capacity: rm.Capacity,
		items:    item.ExtractAll(rm.Items),
	}
}

func ExtractEquipable(rm EquipableRestModel) (EquipableModel, error) {
	es, err := model.SliceMap(equipable.Extract)(model.FixedProvider(rm.Items))(model.ParallelMap())()
	if err != nil {
		return EquipableModel{}, err
	}

	return EquipableModel{
		capacity: rm.Capacity,
		items:    es,
	}, nil
}

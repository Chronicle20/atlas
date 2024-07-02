package inventory

import (
	"atlas-character/equipable"
	"atlas-character/inventory/item"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/manyminds/api2go/jsonapi"
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

func Transform(m Model) (RestModel, error) {
	eqps, err := model.TransformAll(m.Equipable().items, equipable.Transform)
	if err != nil {
		return RestModel{}, err
	}
	stps, err := model.TransformAll(m.Setup().Items(), item.Transform)
	if err != nil {
		return RestModel{}, err
	}
	usps, err := model.TransformAll(m.Useable().Items(), item.Transform)
	if err != nil {
		return RestModel{}, err
	}
	etcs, err := model.TransformAll(m.ETC().Items(), item.Transform)
	if err != nil {
		return RestModel{}, err
	}
	cashs, err := model.TransformAll(m.Cash().Items(), item.Transform)
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Equipable: EquipableRestModel{
			Type:     TypeEquip,
			Capacity: m.equipable.Capacity(),
			Items:    eqps,
		},
		Setup: ItemRestModel{
			Type:     TypeSetup,
			Capacity: m.setup.Capacity(),
			Items:    stps,
		},
		Useable: ItemRestModel{
			Type:     TypeUse,
			Capacity: m.useable.Capacity(),
			Items:    usps,
		},
		Etc: ItemRestModel{
			Type:     TypeETC,
			Capacity: m.etc.Capacity(),
			Items:    etcs,
		},
		Cash: ItemRestModel{
			Type:     TypeCash,
			Capacity: m.cash.Capacity(),
			Items:    cashs,
		},
	}, nil
}

func Extract(m RestModel) (Model, error) {
	equipable, err := model.Transform(m.Equipable, ExtractEquipable)
	if err != nil {
		return Model{}, err
	}
	useable, err := model.Transform(m.Useable, ExtractItem)
	if err != nil {
		return Model{}, err
	}
	setup, err := model.Transform(m.Setup, ExtractItem)
	if err != nil {
		return Model{}, err
	}
	etc, err := model.Transform(m.Etc, ExtractItem)
	if err != nil {
		return Model{}, err
	}
	cash, err := model.Transform(m.Cash, ExtractItem)
	if err != nil {
		return Model{}, err
	}

	return Model{
		equipable: equipable,
		useable:   useable,
		setup:     setup,
		etc:       etc,
		cash:      cash,
	}, nil
}

func ExtractEquipable(m EquipableRestModel) (EquipableModel, error) {
	items, err := model.TransformAll(m.Items, equipable.Extract)
	if err != nil {
		return EquipableModel{}, err
	}

	return EquipableModel{
		capacity: m.Capacity,
		items:    items,
	}, nil
}

func ExtractItem(m ItemRestModel) (ItemModel, error) {
	items, err := model.TransformAll(m.Items, item.Extract)
	if err != nil {
		return ItemModel{}, err
	}
	return ItemModel{
		capacity: m.Capacity,
		items:    items,
	}, nil
}

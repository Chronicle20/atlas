package item

import "strconv"

type RestModel struct {
	Id       uint32 `json:"-"`
	ItemId   uint32 `json:"itemId"`
	Slot     int16  `json:"slot"`
	Quantity uint32 `json:"quantity"`
}

func (r RestModel) GetName() string {
	return "items"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func TransformAll(models []Model) []RestModel {
	rms := make([]RestModel, 0)
	for _, m := range models {
		rms = append(rms, Transform(m))
	}
	return rms
}

func Transform(val Model) RestModel {
	return RestModel{
		Id:       val.id,
		ItemId:   val.itemId,
		Slot:     val.slot,
		Quantity: val.quantity,
	}
}

func ExtractAll(items []RestModel) []Model {
	results := make([]Model, len(items))
	for i, m := range items {
		results[i] = Extract(m)
	}
	return results
}

func Extract(m RestModel) Model {
	return Model{
		id:       m.Id,
		itemId:   m.ItemId,
		slot:     m.Slot,
		quantity: m.Quantity,
	}
}

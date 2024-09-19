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

func Extract(m RestModel) (Model, error) {
	return Model{
		id:       m.Id,
		itemId:   m.ItemId,
		slot:     m.Slot,
		quantity: m.Quantity,
	}, nil
}

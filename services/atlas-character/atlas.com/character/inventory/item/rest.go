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

func Transform(val Model) (RestModel, error) {
	return RestModel{
		Id:       val.id,
		ItemId:   val.itemId,
		Slot:     val.slot,
		Quantity: val.quantity,
	}, nil
}

func Extract(model RestModel) (Model, error) {
	return Model{
		id:       model.Id,
		itemId:   model.ItemId,
		slot:     model.Slot,
		quantity: model.Quantity,
	}, nil
}

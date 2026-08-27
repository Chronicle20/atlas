package item

type RestModel struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint16 `json:"quantity"`
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		itemId:   rm.ItemId,
		quantity: rm.Quantity,
	}, nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		ItemId:   m.itemId,
		Quantity: m.quantity,
	}, nil
}

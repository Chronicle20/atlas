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

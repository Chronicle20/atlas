package equipable

func makeModel(e entity) (Model, error) {
	return Model{
		id:     e.ID,
		itemId: e.ItemId,
		slot:   e.Slot,
	}, nil
}

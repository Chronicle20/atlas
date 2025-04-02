package wishlist

type Model struct {
	id           uint32
	serialNumber uint32
}

func NewModel(id uint32, serialNumber uint32) Model {
	return Model{
		id:           id,
		serialNumber: serialNumber,
	}
}

package pet

type Model struct {
	id     uint64
	itemId uint32
}

func (p Model) Id() uint64 {
	return p.id
}

func (p Model) ItemId() uint32 {
	return p.itemId
}

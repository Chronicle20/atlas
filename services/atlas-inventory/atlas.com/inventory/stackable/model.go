package stackable

type Model struct {
	id           uint32
	quantity     uint32
	ownerId      uint32
	flag         uint16
	rechargeable uint64
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Quantity() uint32 {
	return m.quantity
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}

func (m Model) Flag() uint16 {
	return m.flag
}

func (m Model) Rechargeable() uint64 {
	return m.rechargeable
}

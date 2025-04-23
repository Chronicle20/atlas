package _map

type Model struct {
	returnMapId uint32
}

func (m Model) ReturnMapId() uint32 {
	return m.returnMapId
}

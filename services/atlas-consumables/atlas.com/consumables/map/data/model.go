package data

type Model struct {
	returnMapId uint32
}

func (m Model) ReturnMapId() uint32 {
	return m.returnMapId
}

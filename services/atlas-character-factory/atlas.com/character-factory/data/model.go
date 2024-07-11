package data

type Model struct {
	name string
	wz   string
	slot int16
}

func (m Model) Name() string {
	return m.name
}

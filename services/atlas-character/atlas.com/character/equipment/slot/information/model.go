package information

type Model struct {
	itemId uint32
	name   string
	wz     string
	slot   int16
}

func (m Model) Slot() int16 {
	return m.slot
}

func (m Model) Name() string {
	return m.name
}

package progress

type Model struct {
	id         uint32
	infoNumber uint32
	progress   string
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) InfoNumber() uint32 {
	return m.infoNumber
}

func (m Model) Progress() string {
	return m.progress
}

package map_

type Model struct {
	clock bool
}

func (m Model) Clock() bool {
	return m.clock
}

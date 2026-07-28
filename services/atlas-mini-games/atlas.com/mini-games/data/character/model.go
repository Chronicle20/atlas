package character

// Model is the minimal character view the mini-game validation ladder needs:
// Hp (alive check) and Name.
type Model struct {
	id   uint32
	name string
	hp   uint16
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Hp() uint16 {
	return m.hp
}

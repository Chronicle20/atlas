package character

type Model struct {
	id    uint32
	name  string
	level byte
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Level() byte {
	return m.level
}

// NewModel creates a new character model for testing purposes
func NewModel(id uint32, name string, level byte) Model {
	return Model{
		id:    id,
		name:  name,
		level: level,
	}
}

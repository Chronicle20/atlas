package character

// Model is a character's crafting-relevant attributes, as resolved from
// atlas-character. Level gates a recipe's reqLevel; Meso gates its meso
// cost.
type Model struct {
	id    uint32
	level byte
	meso  uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) Meso() uint32 {
	return m.meso
}

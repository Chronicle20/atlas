package skill

// Model is one of a character's skills, as resolved from atlas-skills. Id is
// the skill id (e.g. the Maker skill); Level gates a recipe's
// reqSkillLevel.
type Model struct {
	id    uint32
	level byte
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Level() byte {
	return m.level
}

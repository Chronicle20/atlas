package quest

// Model is one of a character's quest progress entries, as resolved from
// atlas-quest. QuestId and State are the pair a recipe's reqQuest
// ingredient (design C-5) checks against.
type Model struct {
	id          uint32
	characterId uint32
	questId     uint32
	state       byte
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) QuestId() uint32 {
	return m.questId
}

func (m Model) State() byte {
	return m.state
}

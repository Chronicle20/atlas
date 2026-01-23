package drop

// Model represents a reactor drop entry from drop-information
type Model struct {
	reactorId uint32
	itemId    uint32
	questId   uint32
	chance    uint32
}

func (m Model) ReactorId() uint32 {
	return m.reactorId
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) QuestId() uint32 {
	return m.questId
}

func (m Model) Chance() uint32 {
	return m.chance
}

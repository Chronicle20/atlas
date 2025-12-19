package chair

type Model struct {
	id          uint32
	chairType   string
	characterId uint32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

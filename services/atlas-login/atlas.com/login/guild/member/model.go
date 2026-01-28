package member

type Model struct {
	characterId uint32
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

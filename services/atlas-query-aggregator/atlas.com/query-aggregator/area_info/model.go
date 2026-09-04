package area_info

// Model is the domain model for a character's stored area-info string.
type Model struct {
	characterId uint32
	area        uint16
	info        string
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Area() uint16 {
	return m.area
}

func (m Model) Info() string {
	return m.info
}

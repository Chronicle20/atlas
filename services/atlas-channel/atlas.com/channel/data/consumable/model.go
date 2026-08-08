package consumable

// Model is a consumable's recovery spec, as resolved from atlas-data.
type Model struct {
	id   uint32
	spec map[SpecType]int32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) GetSpec(specType SpecType) (int32, bool) {
	val, ok := m.spec[specType]
	return val, ok
}

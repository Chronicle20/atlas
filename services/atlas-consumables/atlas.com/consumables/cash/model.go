package cash

type Model struct {
	id      uint32
	slotMax uint32
	spec    map[SpecType]int32
}

func (m Model) GetSpec(specType SpecType) (int32, bool) {
	val, ok := m.spec[specType]
	return val, ok
}

func (m Model) Indexes() []uint32 {
	indexes := make([]uint32, 0)
	for _, v := range SpecTypeIndexes {
		if m.spec[v] != 0 {
			indexes = append(indexes, uint32(m.spec[v]))
		}
	}
	return indexes
}

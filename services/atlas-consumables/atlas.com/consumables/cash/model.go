package cash

type Model struct {
	id          uint32
	slotMax     uint32
	spec        map[SpecType]int32
	petSkills   []string
	petSkillAdd bool
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

// PetSkills returns the semantic skill keys this 0519 item grants or removes.
func (m Model) PetSkills() []string { return m.petSkills }

// PetSkillAdd reports grant (true) vs removal (false); only meaningful when
// PetSkills is non-empty.
func (m Model) PetSkillAdd() bool { return m.petSkillAdd }

package equipment

// Model is an equipment template's pet-ability attributes, as resolved from
// atlas-data.
type Model struct {
	id           uint32
	petAbilities []string
}

func (m Model) Id() uint32 {
	return m.id
}

// PetAbilities lists the equip's truthy pet-ability attributes (equip-family
// spelling: consumeHP, consumeMP, sweepForDrop, ...).
func (m Model) PetAbilities() []string {
	return m.petAbilities
}

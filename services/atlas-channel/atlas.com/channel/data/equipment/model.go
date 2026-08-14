package equipment

// Model is an equipment template's pet-ability attributes, as resolved from
// atlas-data.
type Model struct {
	id           uint32
	petAbilities []string
	notExtend    bool
}

func (m Model) Id() uint32 {
	return m.id
}

// PetAbilities lists the equip's truthy pet-ability attributes (equip-family
// spelling: consumeHP, consumeMP, sweepForDrop, ...).
func (m Model) PetAbilities() []string {
	return m.petAbilities
}

// NotExtend reports whether info/notExtend is set on the equip template — a
// WZ blacklist that forbids applying an item-expiration extender (Magical
// Sandglass) to it. The client enforces it via CItemInfo::IsNotExtendItem;
// the server re-checks so a crafted request cannot bypass it.
func (m Model) NotExtend() bool {
	return m.notExtend
}

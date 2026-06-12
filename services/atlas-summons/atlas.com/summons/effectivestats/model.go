package effectivestats

// Model is the immutable representation of a character's session-effective
// combat stats, used by the summon damage ceiling.
type Model struct {
	strength     uint32
	dexterity    uint32
	luck         uint32
	intelligence uint32
	weaponAttack uint32
	magicAttack  uint32
}

func (m Model) Strength() uint32     { return m.strength }
func (m Model) Dexterity() uint32    { return m.dexterity }
func (m Model) Luck() uint32         { return m.luck }
func (m Model) Intelligence() uint32 { return m.intelligence }
func (m Model) WeaponAttack() uint32 { return m.weaponAttack }
func (m Model) MagicAttack() uint32  { return m.magicAttack }

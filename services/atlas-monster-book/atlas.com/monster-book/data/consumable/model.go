package consumable

// Model is the minimal immutable view of an atlas-data consumable that the
// monster-book cover resolver needs: whether the item is a monster-book card
// and, if so, the mob id the card represents.
type Model struct {
	monsterBook bool
	monsterId   uint32
}

func (m Model) MonsterBook() bool { return m.monsterBook }
func (m Model) MonsterId() uint32 { return m.monsterId }

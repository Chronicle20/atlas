package consumable

// Model represents consumable item data from atlas-data
type Model struct {
	id           uint32
	slotMax      uint32
	rechargeable bool
}

func (m Model) Id() uint32          { return m.id }
func (m Model) SlotMax() uint32     { return m.slotMax }
func (m Model) Rechargeable() bool  { return m.rechargeable }
func (m Model) CanMerge() bool      { return !m.rechargeable }

package setup

// Model represents setup item data from atlas-data
type Model struct {
	id      uint32
	slotMax uint32
}

func (m Model) Id() uint32      { return m.id }
func (m Model) SlotMax() uint32 { return m.slotMax }
func (m Model) CanMerge() bool  { return true }

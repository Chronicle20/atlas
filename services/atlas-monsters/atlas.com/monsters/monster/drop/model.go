package drop

type Model struct {
	itemId          uint32
	minimumQuantity uint32
	maximumQuantity uint32
	questId         uint32
	chance          uint32
}

func (m Model) ItemId() uint32          { return m.itemId }
func (m Model) MinimumQuantity() uint32 { return m.minimumQuantity }
func (m Model) MaximumQuantity() uint32 { return m.maximumQuantity }
func (m Model) QuestId() uint32         { return m.questId }
func (m Model) Chance() uint32          { return m.chance }

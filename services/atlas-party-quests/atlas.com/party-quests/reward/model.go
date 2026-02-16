package reward

type WeightedItem struct {
	templateId uint32
	weight     uint32
	quantity   uint32
}

func (w WeightedItem) TemplateId() uint32 { return w.templateId }
func (w WeightedItem) Weight() uint32     { return w.weight }
func (w WeightedItem) Quantity() uint32   { return w.quantity }

type Model struct {
	rewardType string
	amount     uint32
	items      []WeightedItem
}

func (m Model) Type() string          { return m.rewardType }
func (m Model) Amount() uint32        { return m.amount }
func (m Model) Items() []WeightedItem { return m.items }

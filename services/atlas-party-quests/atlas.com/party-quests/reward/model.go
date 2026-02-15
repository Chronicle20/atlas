package reward

import "errors"

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

func (m Model) Type() string            { return m.rewardType }
func (m Model) Amount() uint32          { return m.amount }
func (m Model) Items() []WeightedItem   { return m.items }

type WeightedItemBuilder struct {
	templateId uint32
	weight     uint32
	quantity   uint32
}

func NewWeightedItemBuilder() *WeightedItemBuilder {
	return &WeightedItemBuilder{}
}

func (b *WeightedItemBuilder) SetTemplateId(id uint32) *WeightedItemBuilder {
	b.templateId = id
	return b
}

func (b *WeightedItemBuilder) SetWeight(w uint32) *WeightedItemBuilder {
	b.weight = w
	return b
}

func (b *WeightedItemBuilder) SetQuantity(q uint32) *WeightedItemBuilder {
	b.quantity = q
	return b
}

func (b *WeightedItemBuilder) Build() (WeightedItem, error) {
	if b.templateId == 0 {
		return WeightedItem{}, errors.New("templateId is required")
	}
	return WeightedItem{
		templateId: b.templateId,
		weight:     b.weight,
		quantity:   b.quantity,
	}, nil
}

type Builder struct {
	rewardType string
	amount     uint32
	items      []WeightedItem
}

func NewBuilder() *Builder {
	return &Builder{
		items: make([]WeightedItem, 0),
	}
}

func (b *Builder) SetType(t string) *Builder {
	b.rewardType = t
	return b
}

func (b *Builder) SetAmount(a uint32) *Builder {
	b.amount = a
	return b
}

func (b *Builder) SetItems(items []WeightedItem) *Builder {
	b.items = items
	return b
}

func (b *Builder) AddItem(item WeightedItem) *Builder {
	b.items = append(b.items, item)
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.rewardType == "" {
		return Model{}, errors.New("type is required")
	}
	return Model{
		rewardType: b.rewardType,
		amount:     b.amount,
		items:      b.items,
	}, nil
}

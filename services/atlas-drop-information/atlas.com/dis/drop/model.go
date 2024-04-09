package drop

type Model struct {
	id              uint32
	monsterId       uint32
	itemId          uint32
	minimumQuantity uint32
	maximumQuantity uint32
	chance          uint32
}

func (d Model) MonsterId() uint32 {
	return d.monsterId
}

func (d Model) ItemId() uint32 {
	return d.itemId
}

func (d Model) MinimumQuantity() uint32 {
	return d.minimumQuantity
}

func (d Model) MaximumQuantity() uint32 {
	return d.maximumQuantity
}

func (d Model) Chance() uint32 {
	return d.chance
}

func (d Model) Id() uint32 {
	return d.id
}

type builder struct {
	id              uint32
	monsterId       uint32
	itemId          uint32
	minimumQuantity uint32
	maximumQuantity uint32
	chance          uint32
}

func NewMonsterDropBuilder(id uint32) *builder {
	return &builder{id: id}
}

func (m *builder) SetMonsterId(monsterId uint32) *builder {
	m.monsterId = monsterId
	return m
}

func (m *builder) SetItemId(itemId uint32) *builder {
	m.itemId = itemId
	return m
}

func (m *builder) SetMinimumQuantity(minimumQuantity uint32) *builder {
	m.minimumQuantity = minimumQuantity
	return m
}

func (m *builder) SetMaximumQuantity(maximumQuantity uint32) *builder {
	m.maximumQuantity = maximumQuantity
	return m
}

func (m *builder) SetChance(chance uint32) *builder {
	m.chance = chance
	return m
}

func (m *builder) Build() Model {
	return Model{
		id:              m.id,
		monsterId:       m.monsterId,
		itemId:          m.itemId,
		minimumQuantity: m.minimumQuantity,
		maximumQuantity: m.maximumQuantity,
		chance:          m.chance,
	}
}

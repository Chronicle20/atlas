package drop

import "errors"

type Builder struct {
	itemId          uint32
	minimumQuantity uint32
	maximumQuantity uint32
	questId         uint32
	chance          uint32
}

func NewBuilder() *Builder {
	return &Builder{
		minimumQuantity: 1,
		maximumQuantity: 1,
	}
}

func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
	return b
}

func (b *Builder) SetMinimumQuantity(quantity uint32) *Builder {
	b.minimumQuantity = quantity
	return b
}

func (b *Builder) SetMaximumQuantity(quantity uint32) *Builder {
	b.maximumQuantity = quantity
	return b
}

func (b *Builder) SetQuestId(questId uint32) *Builder {
	b.questId = questId
	return b
}

func (b *Builder) SetChance(chance uint32) *Builder {
	b.chance = chance
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.minimumQuantity > b.maximumQuantity {
		return Model{}, errors.New("minimumQuantity cannot be greater than maximumQuantity")
	}
	return Model{
		itemId:          b.itemId,
		minimumQuantity: b.minimumQuantity,
		maximumQuantity: b.maximumQuantity,
		questId:         b.questId,
		chance:          b.chance,
	}, nil
}

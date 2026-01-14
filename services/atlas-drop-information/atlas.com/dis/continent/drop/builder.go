package drop

import (
	"errors"

	"github.com/google/uuid"
)

type Builder struct {
	tenantId        uuid.UUID
	id              uint32
	continentId     int32
	itemId          uint32
	minimumQuantity uint32
	maximumQuantity uint32
	questId         uint32
	chance          uint32
}

func NewContinentDropBuilder(tenantId uuid.UUID, id uint32) *Builder {
	return &Builder{tenantId: tenantId, id: id}
}

func (b *Builder) SetContinentId(continentId int32) *Builder {
	b.continentId = continentId
	return b
}

func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
	return b
}

func (b *Builder) SetMinimumQuantity(minimumQuantity uint32) *Builder {
	b.minimumQuantity = minimumQuantity
	return b
}

func (b *Builder) SetMaximumQuantity(maximumQuantity uint32) *Builder {
	b.maximumQuantity = maximumQuantity
	return b
}

func (b *Builder) SetChance(chance uint32) *Builder {
	b.chance = chance
	return b
}

func (b *Builder) SetQuestId(questId uint32) *Builder {
	b.questId = questId
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	return Model{
		tenantId:        b.tenantId,
		id:              b.id,
		continentId:     b.continentId,
		itemId:          b.itemId,
		minimumQuantity: b.minimumQuantity,
		maximumQuantity: b.maximumQuantity,
		questId:         b.questId,
		chance:          b.chance,
	}, nil
}

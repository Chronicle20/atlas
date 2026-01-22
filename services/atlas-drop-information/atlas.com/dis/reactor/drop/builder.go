package drop

import (
	"errors"

	"github.com/google/uuid"
)

type Builder struct {
	tenantId  uuid.UUID
	id        uint32
	reactorId uint32
	itemId    uint32
	questId   uint32
	chance    uint32
}

func NewReactorDropBuilder(tenantId uuid.UUID, id uint32) *Builder {
	return &Builder{tenantId: tenantId, id: id}
}

func (b *Builder) SetReactorId(reactorId uint32) *Builder {
	b.reactorId = reactorId
	return b
}

func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
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
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	return Model{
		tenantId:  b.tenantId,
		id:        b.id,
		reactorId: b.reactorId,
		itemId:    b.itemId,
		questId:   b.questId,
		chance:    b.chance,
	}, nil
}

package area_info

import (
	"github.com/google/uuid"
)

type builder struct {
	id          uuid.UUID
	characterId uint32
	area        uint16
	info        string
}

func NewBuilder() *builder {
	return &builder{}
}

func (b *builder) SetId(id uuid.UUID) *builder {
	b.id = id
	return b
}

func (b *builder) SetCharacterId(characterId uint32) *builder {
	b.characterId = characterId
	return b
}

func (b *builder) SetArea(area uint16) *builder {
	b.area = area
	return b
}

func (b *builder) SetInfo(info string) *builder {
	b.info = info
	return b
}

func (b *builder) Build() Model {
	return Model{
		id:          b.id,
		characterId: b.characterId,
		area:        b.area,
		info:        b.info,
	}
}

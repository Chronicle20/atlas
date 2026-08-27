package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type builder struct {
	characterId uint32
	regular     []_map.Id
	vip         []_map.Id
}

func NewBuilder() *builder {
	return &builder{}
}

func (b *builder) SetCharacterId(characterId uint32) *builder {
	b.characterId = characterId
	return b
}

func (b *builder) SetRegular(maps []_map.Id) *builder {
	b.regular = maps
	return b
}

func (b *builder) SetVip(maps []_map.Id) *builder {
	b.vip = maps
	return b
}

func (b *builder) Build() Model {
	return Model{
		characterId: b.characterId,
		regular:     b.regular,
		vip:         b.vip,
	}
}

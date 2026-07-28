package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type modelBuilder struct {
	characterId uint32
	regular     []_map.Id
	vip         []_map.Id
}

func NewBuilder() *modelBuilder {
	return &modelBuilder{}
}

func (b *modelBuilder) SetCharacterId(characterId uint32) *modelBuilder {
	b.characterId = characterId
	return b
}

func (b *modelBuilder) SetRegular(maps []_map.Id) *modelBuilder {
	b.regular = maps
	return b
}

func (b *modelBuilder) SetVip(maps []_map.Id) *modelBuilder {
	b.vip = maps
	return b
}

func (b *modelBuilder) Build() Model {
	return Model{
		characterId: b.characterId,
		regular:     b.regular,
		vip:         b.vip,
	}
}

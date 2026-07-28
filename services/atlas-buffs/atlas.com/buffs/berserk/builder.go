package berserk

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Builder struct {
	worldId        world.Id
	channelId      channel.Id
	channelKnown   bool
	characterId    uint32
	characterLevel byte
	skillLevel     byte
	dirtyAt        time.Time
}

func NewBuilder(worldId world.Id, characterId uint32, skillLevel byte) *Builder {
	return &Builder{
		worldId:     worldId,
		characterId: characterId,
		skillLevel:  skillLevel,
	}
}

func (b *Builder) SetChannel(channelId channel.Id) *Builder {
	b.channelId = channelId
	b.channelKnown = true
	return b
}

func (b *Builder) SetCharacterLevel(level byte) *Builder {
	b.characterLevel = level
	return b
}

func (b *Builder) SetDirtyAt(at time.Time) *Builder {
	b.dirtyAt = at
	return b
}

func (b *Builder) Build() Model {
	return Model{
		worldId:        b.worldId,
		channelId:      b.channelId,
		channelKnown:   b.channelKnown,
		characterId:    b.characterId,
		characterLevel: b.characterLevel,
		skillLevel:     b.skillLevel,
		dirtyAt:        b.dirtyAt,
	}
}

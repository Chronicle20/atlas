package character

import (
	"atlas-buffs/buff"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Builder constructs a character Model, guaranteeing the invariant every
// consumer of Model depends on: buffs is never nil, so callers may index and
// assign into it without a nil-map panic.
type Builder struct {
	worldId     world.Id
	channelId   channel.Id
	characterId uint32
	buffs       map[string]buff.Model
}

func NewBuilder(worldId world.Id, channelId channel.Id, characterId uint32) *Builder {
	return &Builder{
		worldId:     worldId,
		channelId:   channelId,
		characterId: characterId,
		buffs:       make(map[string]buff.Model),
	}
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetChannelId(channelId channel.Id) *Builder {
	b.channelId = channelId
	return b
}

// SetBuffs copies the supplied map, so a later mutation of the caller's map
// cannot reach through into the built Model.
func (b *Builder) SetBuffs(buffs map[string]buff.Model) *Builder {
	nb := make(map[string]buff.Model, len(buffs))
	for k, v := range buffs {
		nb[k] = v
	}
	b.buffs = nb
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}

	buffs := b.buffs
	if buffs == nil {
		buffs = make(map[string]buff.Model)
	}

	return Model{
		worldId:     b.worldId,
		channelId:   b.channelId,
		characterId: b.characterId,
		buffs:       buffs,
	}, nil
}

package character

import (
	"atlas-buffs/buff"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

type Model struct {
	tenant      tenant.Model
	worldId     world.Id
	channelId   channel.Id
	characterId uint32
	buffs       map[int32]buff.Model
}

func (m Model) Buffs() map[int32]buff.Model {
	// Return defensive copy to prevent external mutation
	result := make(map[int32]buff.Model, len(m.buffs))
	for k, v := range m.buffs {
		result[k] = v
	}
	return result
}

func (m Model) Id() uint32 {
	return m.characterId
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

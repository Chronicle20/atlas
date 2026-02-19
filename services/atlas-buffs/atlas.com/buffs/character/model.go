package character

import (
	"atlas-buffs/buff"
	"encoding/json"

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

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		WorldId     world.Id             `json:"worldId"`
		ChannelId   channel.Id           `json:"channelId"`
		CharacterId uint32               `json:"characterId"`
		Buffs       map[int32]buff.Model `json:"buffs"`
	}{
		WorldId:     m.worldId,
		ChannelId:   m.channelId,
		CharacterId: m.characterId,
		Buffs:       m.buffs,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		WorldId     world.Id             `json:"worldId"`
		ChannelId   channel.Id           `json:"channelId"`
		CharacterId uint32               `json:"characterId"`
		Buffs       map[int32]buff.Model `json:"buffs"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.worldId = aux.WorldId
	m.channelId = aux.ChannelId
	m.characterId = aux.CharacterId
	m.buffs = aux.Buffs
	return nil
}

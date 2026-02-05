package guild

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic                    = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestName             = "REQUEST_NAME"
	CommandTypeRequestEmblem           = "REQUEST_EMBLEM"
	CommandTypeRequestDisband          = "REQUEST_DISBAND"
	CommandTypeRequestCapacityIncrease = "REQUEST_CAPACITY_INCREASE"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type RequestNameBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
}

type RequestEmblemBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
}

type RequestDisbandBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
}

type RequestCapacityIncreaseBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
}

package character

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopicChannelChangeRequest = "COMMAND_TOPIC_CHARACTER_CHANNEL_CHANGE_REQUEST"
	CommandChannelChangeRequest         = "CHANNEL_CHANGE_REQUEST"
)

type ChannelChangeRequestCommand struct {
	TransactionId   uuid.UUID  `json:"transactionId"`
	CharacterId     uint32     `json:"characterId"`
	WorldId         world.Id   `json:"worldId"`
	OldChannelId    channel.Id `json:"oldChannelId"`
	TargetChannelId channel.Id `json:"targetChannelId"`
}

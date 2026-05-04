package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopicChannelChangeRequest = "COMMAND_TOPIC_CHARACTER_CHANNEL_CHANGE_REQUEST"
	CommandChannelChangeRequest         = "CHANNEL_CHANGE_REQUEST"
)

// ChannelChangeRequestCommand mirrors the struct emitted by atlas-channel on
// the EnvCommandTopicChannelChangeRequest topic. atlas-maps redefines the type
// here (rather than importing atlas-channel) to avoid a cross-service Go
// module dependency. JSON tags MUST match atlas-channel's struct exactly.
type ChannelChangeRequestCommand struct {
	TransactionId   uuid.UUID  `json:"transactionId"`
	CharacterId     uint32     `json:"characterId"`
	WorldId         world.Id   `json:"worldId"`
	OldChannelId    channel.Id `json:"oldChannelId"`
	TargetChannelId channel.Id `json:"targetChannelId"`
}

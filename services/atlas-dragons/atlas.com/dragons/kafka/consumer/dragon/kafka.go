package dragon

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// EnvCommandTopic is the COMMAND_TOPIC_DRAGON env var (channel -> dragons).
// The channel-side mirror at
// services/atlas-channel/.../kafka/message/dragon/kafka.go must keep every json
// tag byte-for-byte identical to these definitions.
const EnvCommandTopic = "COMMAND_TOPIC_DRAGON"

const (
	CommandTypeCreate  = "CREATE"
	CommandTypeDestroy = "DESTROY"
	CommandTypeMove    = "MOVE"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CreateCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

type DestroyCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

// MoveCommandBody carries the raw CMovePath blob plus the start position lifted
// from its first four bytes. The serverbound packet has no identity field, so
// CharacterId is the SENDING SESSION's character id, filled in channel-side.
type MoveCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	StartX      int16  `json:"startX"`
	StartY      int16  `json:"startY"`
	Stance      byte   `json:"stance"`
	RawMovement []byte `json:"rawMovement"`
}

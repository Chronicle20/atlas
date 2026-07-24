package broadcast

import (
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	EnvCommandTopicWorldBroadcast = "COMMAND_TOPIC_WORLD_BROADCAST"
)

// EnqueueCommand requests that a megaphone/Maple TV broadcast be appended to
// the (WorldId, Family) queue. TvMessageType is a semantic key
// (NORMAL|STAR|HEART), never a client wire byte — DOM-25(c): a domain
// service must not emit a client-interpreted byte; that resolution happens
// at the packet layer from the tenant messageTypes writer table.
//
// This struct is byte-for-byte identical (JSON shape) to the world-side
// EnqueueCommand at
// services/atlas-world/atlas.com/world/kafka/message/broadcast/kafka.go.
// Cross-service duplication of Kafka message structs is the established
// repo convention (see kafka/message/gachapon/kafka.go, which exists
// identically in both atlas-channel and atlas-saga-orchestrator) — do not
// try to share this type across services.
type EnqueueCommand struct {
	Family          string                     `json:"family"`
	WorldId         byte                       `json:"worldId"`
	ChannelId       byte                       `json:"channelId"`
	CharacterId     uint32                     `json:"characterId"`
	SenderName      string                     `json:"senderName"`
	SenderMedal     string                     `json:"senderMedal"`
	Messages        []string                   `json:"messages"`
	WhispersOn      bool                       `json:"whispersOn"`
	ItemId          uint32                     `json:"itemId"`
	TvMessageType   string                     `json:"tvMessageType"` // A1 delta: semantic key (DOM-25(c) — a domain service must not emit a client byte)
	DurationSeconds uint32                     `json:"durationSeconds"`
	SenderLook      sharedsaga.AvatarSnapshot  `json:"senderLook"`
	ReceiverName    string                     `json:"receiverName"`
	ReceiverLook    *sharedsaga.AvatarSnapshot `json:"receiverLook,omitempty"`
}

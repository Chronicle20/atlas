package worldbroadcast

import (
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	EnvEventTopicWorldBroadcastStatus = "EVENT_TOPIC_WORLD_BROADCAST_STATUS"

	StatusTypeQueued  = "QUEUED"
	StatusTypeStarted = "STARTED"
	StatusTypeEnded   = "ENDED"
)

// StatusEvent: QUEUED carries WaitSeconds; STARTED carries the full render
// payload + TotalWaitSeconds (SEND_TV totalWaitTime); ENDED carries only
// Family/WorldId (+CharacterId of the ended entry).
//
// Mirrors atlas-world's kafka/message/broadcast/kafka.go StatusEvent
// (task-123 Task 8) byte-for-byte: same field set, same json tags, same
// EnvEventTopicWorldBroadcastStatus/StatusType* values.
//
// A1 delta: TvMessageType is a semantic key (NORMAL|STAR|HEART), never a
// client wire byte - DOM-25(c): a domain service must not emit a
// client-interpreted byte; that resolution happens at the packet layer
// from the tenant messageTypes writer table.
type StatusEvent struct {
	Type             string                     `json:"type"`
	Family           string                     `json:"family"`
	WorldId          byte                       `json:"worldId"`
	CharacterId      uint32                     `json:"characterId"`
	WaitSeconds      uint32                     `json:"waitSeconds"`
	TotalWaitSeconds uint32                     `json:"totalWaitSeconds"`
	ChannelId        byte                       `json:"channelId"`
	SenderName       string                     `json:"senderName"`
	SenderMedal      string                     `json:"senderMedal"`
	Messages         []string                   `json:"messages"`
	WhispersOn       bool                       `json:"whispersOn"`
	ItemId           uint32                     `json:"itemId"`
	TvMessageType    string                     `json:"tvMessageType"` // A1 delta: semantic key (DOM-25(c))
	SenderLook       sharedsaga.AvatarSnapshot  `json:"senderLook"`
	ReceiverName     string                     `json:"receiverName"`
	ReceiverLook     *sharedsaga.AvatarSnapshot `json:"receiverLook,omitempty"`
}

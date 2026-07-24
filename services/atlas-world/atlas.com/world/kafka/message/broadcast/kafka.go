package broadcast

import (
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	EnvCommandTopicWorldBroadcast     = "COMMAND_TOPIC_WORLD_BROADCAST"
	EnvEventTopicWorldBroadcastStatus = "EVENT_TOPIC_WORLD_BROADCAST_STATUS"

	StatusTypeQueued  = "QUEUED"
	StatusTypeStarted = "STARTED"
	StatusTypeEnded   = "ENDED"
)

// EnqueueCommand requests that a megaphone/Maple TV broadcast be appended to
// the (WorldId, Family) queue. TvMessageType is a semantic key
// (NORMAL|STAR|HEART), never a client wire byte — DOM-25(c): a domain
// service must not emit a client-interpreted byte; that resolution happens
// at the packet layer from the tenant messageTypes writer table.
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

// StartedPayload carries the render payload + duration needed to build a
// STARTED StatusEvent. Deliberately defined here (in the message package,
// which depends on nothing but sharedsaga) rather than accepting the
// domain broadcast.Entry type directly: kafka/producer/broadcast's only
// caller is broadcast/processor.go (same package as Entry), so a producer
// function that took Entry directly would create an import cycle
// (domain -> producer -> domain). StartedPayload breaks that cycle while
// still letting the processor build STARTED events from an activated Entry.
type StartedPayload struct {
	CharacterId     uint32
	DurationSeconds uint32
	ChannelId       byte
	SenderName      string
	SenderMedal     string
	Messages        []string
	WhispersOn      bool
	ItemId          uint32
	TvMessageType   string
	SenderLook      sharedsaga.AvatarSnapshot
	ReceiverName    string
	ReceiverLook    *sharedsaga.AvatarSnapshot
}

// StatusEvent: QUEUED carries WaitSeconds; STARTED carries the full render
// payload + TotalWaitSeconds (SEND_TV totalWaitTime); ENDED carries only
// Family/WorldId (+CharacterId of the ended entry).
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

package megaphone

import (
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	EnvEventTopicMegaphone = "EVENT_TOPIC_MEGAPHONE"
)

// BroadcastEvent is the event fired for the stateless megaphone tiers
// (MEGAPHONE/SUPER/ITEM/TRIPLE). Unlike the TV/AVATAR family (see
// kafka/message/broadcast), these tiers are not queued — atlas-channel
// consumes this event and renders the broadcast to the target scope
// (channel or world) immediately.
type BroadcastEvent struct {
	Tier        string                    `json:"tier"`
	Scope       string                    `json:"scope"`
	WorldId     byte                      `json:"worldId"`
	ChannelId   byte                      `json:"channelId"`
	CharacterId uint32                    `json:"characterId"`
	SenderName  string                    `json:"senderName"`
	SenderMedal string                    `json:"senderMedal"`
	Messages    []string                  `json:"messages"`
	WhispersOn  bool                      `json:"whispersOn"`
	Item        *sharedsaga.AssetSnapshot `json:"item,omitempty"`
}

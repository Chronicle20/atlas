package incubator

const (
	EnvEventTopicIncubatorResult = "EVENT_TOPIC_INCUBATOR_RESULT"
)

// ResultEvent delivers the outcome of an incubator use to the channel, which
// announces the result via a packet. CharacterId/WorldId/ChannelId identify
// where to route the announcement; ItemId/Count describe the resulting item
// (ItemId 0 signals a failed/empty result).
type ResultEvent struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	ItemId      uint32 `json:"itemId"`
	Count       uint32 `json:"count"`
}

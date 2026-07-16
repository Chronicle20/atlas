package incubator

const (
	EnvEventTopicIncubatorResult = "EVENT_TOPIC_INCUBATOR_RESULT"
)

// ResultEvent delivers the outcome of an incubator use to the channel, which
// announces the result via a packet. CharacterId/WorldId/ChannelId identify
// where to route the announcement; ItemId/Count describe the resulting item
// (ItemId 0 signals a failed/empty result). EggId is the sacrificed Pigmy
// Egg id; the v95 client uses it to pick the region success NPC.
type ResultEvent struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	ItemId      uint32 `json:"itemId"`
	Count       uint32 `json:"count"`
	EggId       uint32 `json:"eggId"`
}

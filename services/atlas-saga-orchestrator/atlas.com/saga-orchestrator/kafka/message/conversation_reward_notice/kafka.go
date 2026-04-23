package conversation_reward_notice

const (
	EnvEventTopic = "EVENT_TOPIC_CONVERSATION_REWARD_NOTICE"

	KindItemGain = "item_gain"
	KindItemLoss = "item_loss"
)

// EventBody is emitted by atlas-saga-orchestrator on successful completion of a
// conversation-sourced item gain/loss step (where ShowEffect was set on the
// originating saga payload). atlas-channel consumes it and renders the
// appropriate v83 client packet (item-gain effect or item-loss chat line).
type EventBody struct {
	CharacterId uint32 `json:"characterId"`
	Kind        string `json:"kind"`
	ItemId      uint32 `json:"itemId"`
	Quantity    uint32 `json:"quantity"`
}

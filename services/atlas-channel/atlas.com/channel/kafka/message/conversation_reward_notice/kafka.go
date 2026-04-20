package conversation_reward_notice

const (
	EnvEventTopic = "EVENT_TOPIC_CONVERSATION_REWARD_NOTICE"

	KindItemGain = "item_gain"
	KindItemLoss = "item_loss"
)

// EventBody is the message produced by atlas-saga-orchestrator when a
// conversation-sourced AwardAsset / DestroyAsset / DestroyAssetFromSlot step
// completes with ShowEffect=true. atlas-channel renders the gain effect or
// loss chat line on the target session.
type EventBody struct {
	CharacterId uint32 `json:"characterId"`
	Kind        string `json:"kind"`
	ItemId      uint32 `json:"itemId"`
	Quantity    uint32 `json:"quantity"`
}

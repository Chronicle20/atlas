package gachapon

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopicGachaponRewardWon topic.Token = "EVENT_TOPIC_GACHAPON_REWARD_WON"
)

type RewardWonEvent struct {
	CharacterId  uint32 `json:"characterId"`
	WorldId      byte   `json:"worldId"`
	ItemId       uint32 `json:"itemId"`
	Quantity     uint32 `json:"quantity"`
	Tier         string `json:"tier"`
	GachaponId   string `json:"gachaponId"`
	GachaponName string `json:"gachaponName"`
	AssetId      uint32 `json:"assetId"`
}

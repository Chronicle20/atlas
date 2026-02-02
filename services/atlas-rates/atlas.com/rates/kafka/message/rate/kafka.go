package rate

const (
	EnvEventTopicWorldRate = "EVENT_TOPIC_WORLD_RATE"

	TypeRateChanged = "RATE_CHANGED"
)

type RateType string

const (
	RateTypeExp      RateType = "exp"
	RateTypeMeso     RateType = "meso"
	RateTypeItemDrop RateType = "item_drop"
	RateTypeQuestExp RateType = "quest_exp"
)

type WorldRateEvent struct {
	Type       string   `json:"type"`
	WorldId    byte     `json:"worldId"`
	RateType   RateType `json:"rateType"`
	Multiplier float64  `json:"multiplier"`
}

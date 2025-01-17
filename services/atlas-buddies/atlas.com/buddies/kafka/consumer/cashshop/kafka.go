package cashshop

const (
	EnvEventTopicCashShopStatus           = "EVENT_TOPIC_CASH_SHOP_STATUS"
	EventCashShopStatusTypeCharacterEnter = "CHARACTER_ENTER"
	EventCashShopStatusTypeCharacterExit  = "CHARACTER_EXIT"
)

type statusEvent[E any] struct {
	WorldId byte   `json:"worldId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type characterMovementBody struct {
	CharacterId uint32 `json:"characterId"`
}

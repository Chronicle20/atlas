package cashshop

import "github.com/Chronicle20/atlas-constants/world"

const (
	EnvEventTopicStatus           = "EVENT_TOPIC_CASH_SHOP_STATUS"
	EventStatusTypeCharacterEnter = "CHARACTER_ENTER"
	EventStatusTypeCharacterExit  = "CHARACTER_EXIT"
)

type StatusEvent[E any] struct {
	WorldId world.Id `json:"worldId"`
	Type    string   `json:"type"`
	Body    E        `json:"body"`
}

type MovementBody struct {
	CharacterId uint32 `json:"characterId"`
}

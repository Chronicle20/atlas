package cashshop

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicStatus           topic.Token = "EVENT_TOPIC_CASH_SHOP_STATUS"
	EventStatusTypeCharacterEnter             = "CHARACTER_ENTER"
	EventStatusTypeCharacterExit              = "CHARACTER_EXIT"
)

type StatusEvent[E any] struct {
	WorldId world.Id `json:"worldId"`
	Type    string   `json:"type"`
	Body    E        `json:"body"`
}

type MovementBody struct {
	CharacterId uint32 `json:"characterId"`
}

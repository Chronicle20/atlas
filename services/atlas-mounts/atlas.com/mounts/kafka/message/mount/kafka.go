package mount

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvStatusEventTopic = "EVENT_TOPIC_MOUNT_STATUS"
	StatusEventTypeSet  = "SET"
	StatusEventTypeTick = "TICK"
	StatusEventTypeFeed = "FEED"
)

type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type StatusEventBody struct {
	Level     int  `json:"level"`
	Exp       int  `json:"exp"`
	Tiredness int  `json:"tiredness"`
	LevelUp   bool `json:"levelUp"`
	TooTired  bool `json:"tooTired"`
}

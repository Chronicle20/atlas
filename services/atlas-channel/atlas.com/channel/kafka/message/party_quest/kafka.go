package party_quest

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventStatusTopic = "EVENT_TOPIC_PARTY_QUEST_STATUS"

	EventTypeStageCleared = "STAGE_CLEARED"
)

type StatusEvent[E any] struct {
	WorldId    world.Id  `json:"worldId"`
	InstanceId uuid.UUID `json:"instanceId"`
	QuestId    string    `json:"questId"`
	Type       string    `json:"type"`
	Body       E         `json:"body"`
}

type StageClearedEventBody struct {
	StageIndex     uint32      `json:"stageIndex"`
	ChannelId      channel.Id  `json:"channelId"`
	MapIds         []uint32    `json:"mapIds"`
	FieldInstances []uuid.UUID `json:"fieldInstances"`
}

package quest

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic                   = "COMMAND_TOPIC_QUEST_CONVERSATION"
	CommandTypeStartQuestConversation = "START_QUEST_CONVERSATION"
)

type Command[E any] struct {
	QuestId     uint32 `json:"questId"`
	NpcId       uint32 `json:"npcId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StartQuestConversationCommandBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
}

// Status event types for quest status changes from atlas-quest service
const (
	EnvStatusEventTopic              = "EVENT_TOPIC_QUEST_STATUS"
	StatusEventTypeStarted           = "STARTED"
	StatusEventTypeCompleted         = "COMPLETED"
	StatusEventTypeForfeited         = "FORFEITED"
	StatusEventTypeProgressUpdated   = "PROGRESS_UPDATED"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type QuestStartedEventBody struct {
	QuestId uint32 `json:"questId"`
}

type QuestCompletedEventBody struct {
	QuestId uint32 `json:"questId"`
}

type QuestForfeitedEventBody struct {
	QuestId uint32 `json:"questId"`
}

type QuestProgressUpdatedEventBody struct {
	QuestId  uint32 `json:"questId"`
	Progress string `json:"progress"`
}

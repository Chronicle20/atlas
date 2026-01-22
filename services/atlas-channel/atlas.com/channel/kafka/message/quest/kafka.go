package quest

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic                   = "COMMAND_TOPIC_QUEST_CONVERSATION"
	CommandTypeStartQuestConversation = "START_QUEST_CONVERSATION"
)

// Quest service command topic and types (for atlas-quest)
const (
	EnvQuestCommandTopic         = "COMMAND_TOPIC_QUEST"
	QuestCommandTypeStart        = "START"
	QuestCommandTypeComplete     = "COMPLETE"
	QuestCommandTypeForfeit      = "FORFEIT"
	QuestCommandTypeRestoreItem  = "RESTORE_ITEM"
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

// QuestCommand represents a command message for the atlas-quest service
type QuestCommand[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// StartQuestCommandBody is the body for starting a quest
type StartQuestCommandBody struct {
	QuestId uint32 `json:"questId"`
	NpcId   uint32 `json:"npcId,omitempty"`
	Force   bool   `json:"force,omitempty"` // If true, skip requirement checks
}

// CompleteQuestCommandBody is the body for completing a quest
type CompleteQuestCommandBody struct {
	QuestId   uint32 `json:"questId"`
	NpcId     uint32 `json:"npcId,omitempty"`
	Selection int32  `json:"selection,omitempty"`
	Force     bool   `json:"force,omitempty"`
}

// ForfeitQuestCommandBody is the body for forfeiting a quest
type ForfeitQuestCommandBody struct {
	QuestId uint32 `json:"questId"`
}

// RestoreItemCommandBody is the body for restoring a lost quest item
type RestoreItemCommandBody struct {
	QuestId uint32 `json:"questId"`
	ItemId  uint32 `json:"itemId"`
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
	QuestId  uint32 `json:"questId"`
	Progress string `json:"progress"`
}

type QuestCompletedEventBody struct {
	QuestId     uint32    `json:"questId"`
	CompletedAt time.Time `json:"completedAt"`
}

type QuestForfeitedEventBody struct {
	QuestId uint32 `json:"questId"`
}

type QuestProgressUpdatedEventBody struct {
	QuestId  uint32 `json:"questId"`
	Progress string `json:"progress"`
}

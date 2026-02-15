package party_quest

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_PARTY_QUEST"

	CommandTypeRegister         = "REGISTER"
	CommandTypeStart            = "START"
	CommandTypeStageClearAttempt = "STAGE_CLEAR_ATTEMPT"
	CommandTypeStageAdvance     = "STAGE_ADVANCE"
	CommandTypeForfeit          = "FORFEIT"
	CommandTypeUpdateStageState = "UPDATE_STAGE_STATE"

	EnvEventStatusTopic = "EVENT_TOPIC_PARTY_QUEST_STATUS"

	EventTypeInstanceCreated    = "INSTANCE_CREATED"
	EventTypeRegistrationOpened = "REGISTRATION_OPENED"
	EventTypeStarted            = "STARTED"
	EventTypeStageCleared       = "STAGE_CLEARED"
	EventTypeStageAdvanced      = "STAGE_ADVANCED"
	EventTypeCompleted          = "COMPLETED"
	EventTypeFailed             = "FAILED"
	EventTypeInstanceDestroyed  = "INSTANCE_DESTROYED"
)

type Command[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type RegisterCommandBody struct {
	QuestId string `json:"questId"`
	PartyId uint32 `json:"partyId,omitempty"`
}

type StartCommandBody struct {
	InstanceId uuid.UUID `json:"instanceId"`
}

type StageClearAttemptCommandBody struct {
	InstanceId uuid.UUID `json:"instanceId"`
}

type StageAdvanceCommandBody struct {
	InstanceId uuid.UUID `json:"instanceId"`
}

type ForfeitCommandBody struct {
	InstanceId uuid.UUID `json:"instanceId"`
}

type UpdateStageStateCommandBody struct {
	InstanceId   uuid.UUID         `json:"instanceId"`
	ItemCounts   map[uint32]uint32 `json:"itemCounts,omitempty"`
	MonsterKills map[uint32]uint32 `json:"monsterKills,omitempty"`
}

type StatusEvent[E any] struct {
	WorldId    world.Id  `json:"worldId"`
	InstanceId uuid.UUID `json:"instanceId"`
	QuestId    string    `json:"questId"`
	Type       string    `json:"type"`
	Body       E         `json:"body"`
}

type InstanceCreatedEventBody struct {
	PartyId   uint32 `json:"partyId"`
	ChannelId byte   `json:"channelId"`
}

type RegistrationOpenedEventBody struct {
	Duration int64 `json:"duration"`
}

type StartedEventBody struct {
	StageIndex uint32   `json:"stageIndex"`
	MapIds     []uint32 `json:"mapIds"`
}

type StageClearedEventBody struct {
	StageIndex uint32 `json:"stageIndex"`
}

type StageAdvancedEventBody struct {
	StageIndex uint32   `json:"stageIndex"`
	MapIds     []uint32 `json:"mapIds"`
}

type CompletedEventBody struct {
}

type FailedEventBody struct {
	Reason string `json:"reason"`
}

type InstanceDestroyedEventBody struct {
}

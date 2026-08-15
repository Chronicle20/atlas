package npc

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic                 = "COMMAND_TOPIC_NPC"
	CommandTypeStartConversation    = "START_CONVERSATION"
	CommandTypeContinueConversation = "CONTINUE_CONVERSATION"
	CommandTypeEndConversation      = "END_CONVERSATION"

	// CommandTypeStartItemConversation opens a scripted item's own dialogue
	// (the 243xxxx family). Unlike START_CONVERSATION the conversation is keyed
	// by item id, not by NPC — NpcId carries only the avatar it renders with.
	CommandTypeStartItemConversation = "START_ITEM_CONVERSATION"

	EnvConversationCommandTopic = "COMMAND_TOPIC_NPC_CONVERSATION"
	CommandTypeSimple           = "SIMPLE"
	CommandTypeText             = "TEXT"
	CommandTypeStyle            = "STYLE"
	CommandTypeNumber           = "NUMBER"
	CommandTypeSlideMenu        = "SLIDE_MENU"
)

type Command[E any] struct {
	// TransactionId correlates this command with the saga step that issued it.
	// uuid.Nil means "not saga-driven" — the ordinary NPC-talk path — and the
	// handler emits no status event for it. Non-nil means a saga step is
	// awaiting STARTED or START_ERROR on EnvStatusEventTopic.
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
	NpcId         uint32    `json:"npcId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CommandConversationStartBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	AccountId uint32     `json:"accountId"`
}

// CommandItemConversationStartBody starts a scripted item's dialogue. Slot is
// carried so the destroy step's payload and this command describe the same
// asset; the conversation itself does not consume.
type CommandItemConversationStartBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	AccountId uint32     `json:"accountId"`
	ItemId    uint32     `json:"itemId"`
	Slot      int16      `json:"slot"`
}

type CommandConversationContinueBody struct {
	Action          byte  `json:"action"`
	LastMessageType byte  `json:"lastMessageType"`
	Selection       int32 `json:"selection"`
}

type CommandConversationEndBody struct{}

type ConversationCommand[E any] struct {
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	MapId          _map.Id    `json:"mapId"`
	Instance       uuid.UUID  `json:"instance"`
	CharacterId    uint32     `json:"characterId"`
	NpcId          uint32     `json:"npcId"`
	Speaker        string     `json:"speaker"`
	EndChat        bool       `json:"endChat"`
	SecondaryNpcId uint32     `json:"secondaryNpcId"`
	Message        string     `json:"message"`
	Type           string     `json:"type"`
	Body           E          `json:"body"`
}

type CommandSimpleBody struct {
	Type string `json:"type"`
}

type CommandNumberBody struct {
	DefaultValue uint32 `json:"defaultValue"`
	MinValue     uint32 `json:"minValue"`
	MaxValue     uint32 `json:"maxValue"`
}

type CommandStyleBody struct {
	Styles []uint32 `json:"styles"`
}

type CommandSlideMenuBody struct {
	MenuType uint32 `json:"menuType"`
}

const (
	EnvEventTopicCharacterStatus        = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"
)

type StatusEvent[E any] struct {
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	WorldId     world.Id `json:"worldId"`
	Body        E        `json:"body"`
}

type StatusEventStatChangedBody struct {
	ChannelId       channel.Id             `json:"channelId"`
	ExclRequestSent bool                   `json:"exclRequestSent"`
	Updates         []stat.Type            `json:"updates"`
	Values          map[string]interface{} `json:"values,omitempty"`
}

const (
	// EnvStatusEventTopic reports the outcome of a saga-driven conversation
	// start. atlas-npc-conversations produced no status topic before task-230;
	// it only consumed EVENT_TOPIC_SAGA_STATUS for sagas a conversation
	// initiates. The awaited-step saga needs the opposite direction.
	EnvStatusEventTopic = "EVENT_TOPIC_NPC_CONVERSATION_STATUS"

	StatusEventTypeStarted = "STARTED"

	// StatusEventTypeStartError is deliberately a distinct type rather than a
	// generic ERROR, for the same reason npc-shops splits ENTER_ERROR from
	// ERROR: a generic error type is rendered differently by the channel and
	// would be ambiguous here.
	StatusEventTypeStartError = "START_ERROR"

	// Reasons carried by StatusEventStartErrorBody. The reason is what makes a
	// Loki trace of a content gap distinguishable from a real fault without
	// reading code.
	StartErrorNoConversationAuthored = "NO_CONVERSATION_AUTHORED"
	StartErrorConversationInProgress = "CONVERSATION_IN_PROGRESS"
	StartErrorInternal               = "INTERNAL_ERROR"
)

// ConversationStatusEvent reports a conversation start outcome back to the
// saga that asked for it.
//
// Named ConversationStatusEvent rather than the brief's literal StatusEvent:
// this package already declares a generic `StatusEvent[E any]` above (line
// ~87) for the pre-existing STAT_CHANGED character-status event produced by
// npc/producer.go. A second `type StatusEvent[E any]` in the same package is
// a duplicate-declaration compile error, not a stylistic choice — verified by
// reading this file before writing (CLAUDE.md "verify, don't invent"). Task
// 12's mirror into atlas-saga-orchestrator must reproduce this same name.
type ConversationStatusEvent[E any] struct {
	// TransactionId echoes the originating command's id so a saga can accept
	// only its own event.
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventStartedBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
	// SourceId is the item id for an item conversation, the NPC template id for
	// an NPC conversation — mirroring ConversationContext.SourceId.
	SourceId uint32 `json:"sourceId"`
}

type StatusEventStartErrorBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
	SourceId      uint32 `json:"sourceId"`
	Reason        string `json:"reason"`
}

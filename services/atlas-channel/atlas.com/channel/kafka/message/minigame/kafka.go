package minigame

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// Command envelope. Mirrored byte-for-byte by atlas-channel (task-17); struct
// names, field names and json tags must match this file exactly.
const (
	EnvCommandTopic = "COMMAND_TOPIC_MINI_GAME"

	CommandTypeCreate              = "CREATE"
	CommandTypeVisit               = "VISIT"
	CommandTypeLeave               = "LEAVE"
	CommandTypeChat                = "CHAT"
	CommandTypeReady               = "READY"
	CommandTypeUnready             = "UNREADY"
	CommandTypeStart               = "START"
	CommandTypeMoveStone           = "MOVE_STONE"
	CommandTypeFlipCard            = "FLIP_CARD"
	CommandTypeRequestTie          = "REQUEST_TIE"
	CommandTypeAnswerTie           = "ANSWER_TIE"
	CommandTypeGiveUp              = "GIVE_UP"
	CommandTypeRequestRetreat      = "REQUEST_RETREAT"
	CommandTypeAnswerRetreat       = "ANSWER_RETREAT"
	CommandTypeExpel               = "EXPEL"
	CommandTypeSkip                = "SKIP"
	CommandTypeExitAfterGame       = "EXIT_AFTER_GAME"
	CommandTypeCancelExitAfterGame = "CANCEL_EXIT_AFTER_GAME"
)

type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type CreateCommandBody struct {
	RoomType  byte   `json:"roomType"`
	Title     string `json:"title"`
	Private   bool   `json:"private"`
	Password  string `json:"password"`
	PieceType byte   `json:"pieceType"`
}

type VisitCommandBody struct {
	RoomId   uint32 `json:"roomId"`
	Password string `json:"password"`
}

type ChatCommandBody struct {
	Message string `json:"message"`
}

type MoveStoneCommandBody struct {
	X         uint32 `json:"x"`
	Y         uint32 `json:"y"`
	StoneType byte   `json:"stoneType"`
}

type FlipCardCommandBody struct {
	First     bool `json:"first"`
	CardIndex byte `json:"cardIndex"`
}

// AnswerCommandBody backs both ANSWER_TIE and ANSWER_RETREAT.
type AnswerCommandBody struct {
	Accept bool `json:"accept"`
}

type EmptyCommandBody struct {
}

// StatusEvent envelope. Every event populates RoomId/OwnerId/VisitorId;
// CharacterId is the acting character.
const (
	EnvEventTopicStatus = "EVENT_TOPIC_MINI_GAME_STATUS"

	EventTypeCreated          = "CREATED"
	EventTypeCreateError      = "CREATE_ERROR"
	EventTypeEntered          = "ENTERED"
	EventTypeEnterError       = "ENTER_ERROR"
	EventTypeLeft             = "LEFT"
	EventTypeRoomClosed       = "ROOM_CLOSED"
	EventTypeChat             = "CHAT"
	EventTypeReady            = "READY"
	EventTypeUnready          = "UNREADY"
	EventTypeStarted          = "STARTED"
	EventTypeStonePlaced      = "STONE_PLACED"
	EventTypeCardFlipped      = "CARD_FLIPPED"
	EventTypeTieRequested     = "TIE_REQUESTED"
	EventTypeTieAnswered      = "TIE_ANSWERED"
	EventTypeRetreatRequested = "RETREAT_REQUESTED"
	EventTypeRetreatAnswered  = "RETREAT_ANSWERED"
	EventTypeSkipped          = "SKIPPED"
	EventTypeGameEnded        = "GAME_ENDED"
	EventTypeBalloonUpdated   = "BALLOON_UPDATED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	RoomId        uint32     `json:"roomId"`
	OwnerId       uint32     `json:"ownerId"`
	VisitorId     uint32     `json:"visitorId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type RecordBody struct {
	GameType string `json:"gameType"`
	Wins     uint32 `json:"wins"`
	Ties     uint32 `json:"ties"`
	Losses   uint32 `json:"losses"`
}

type CreatedEventBody struct {
	RoomType    byte       `json:"roomType"`
	Title       string     `json:"title"`
	PieceType   byte       `json:"pieceType"`
	OwnerRecord RecordBody `json:"ownerRecord"`
}

// ErrorEventBody carries the enterError KEY string (e.g. "NOT_WHEN_DEAD"),
// resolved to a numeric code by the channel via the tenant enterError table.
type ErrorEventBody struct {
	Code string `json:"code"`
}

type EnteredEventBody struct {
	Slot          byte       `json:"slot"`
	RoomType      byte       `json:"roomType"`
	Title         string     `json:"title"`
	PieceType     byte       `json:"pieceType"`
	OwnerRecord   RecordBody `json:"ownerRecord"`
	VisitorRecord RecordBody `json:"visitorRecord"`
	OwnerScore    int32      `json:"ownerScore"`
	VisitorScore  int32      `json:"visitorScore"`
}

// LeftEventBody status: 3 closed / 4 left / 5 expelled.
type LeftEventBody struct {
	Slot   byte `json:"slot"`
	Status byte `json:"status"`
}

type RoomClosedEventBody struct {
	VisitorStatus byte `json:"visitorStatus"`
}

type ChatEventBody struct {
	Slot    byte   `json:"slot"`
	Message string `json:"message"`
}

type EmptyEventBody struct {
}

// StartedEventBody Deck is empty for omok.
type StartedEventBody struct {
	RoomType   byte     `json:"roomType"`
	FirstMover byte     `json:"firstMover"`
	Deck       []uint32 `json:"deck"`
}

type StonePlacedEventBody struct {
	X         uint32 `json:"x"`
	Y         uint32 `json:"y"`
	StoneType byte   `json:"stoneType"`
}

type CardFlippedEventBody struct {
	SecondFlip bool `json:"secondFlip"`
	Slot       byte `json:"slot"`
	FirstSlot  byte `json:"firstSlot"`
	ResultType byte `json:"resultType"`
}

type AnswerEventBody struct {
	Accept bool `json:"accept"`
}

type SkippedEventBody struct {
	Who byte `json:"who"`
}

// GameEndedEventBody ResultType 0 win/1 tie/2 forfeit; WinnerSlot 0 owner/1 visitor.
type GameEndedEventBody struct {
	ResultType    byte       `json:"resultType"`
	WinnerSlot    byte       `json:"winnerSlot"`
	OwnerRecord   RecordBody `json:"ownerRecord"`
	VisitorRecord RecordBody `json:"visitorRecord"`
	OwnerScore    int32      `json:"ownerScore"`
	VisitorScore  int32      `json:"visitorScore"`
}

type BalloonEventBody struct {
	Remove      bool   `json:"remove"`
	RoomType    byte   `json:"roomType"`
	Title       string `json:"title"`
	HasPassword bool   `json:"hasPassword"`
	PieceType   byte   `json:"pieceType"`
	Occupancy   byte   `json:"occupancy"`
	InProgress  bool   `json:"inProgress"`
}

package character

const (
	EnvEventTopicCharacterStatus  = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeCreated        = "CREATED"
	StatusEventTypeLogin          = "LOGIN"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
	StatusEventTypeMapChanged     = "MAP_CHANGED"
	StatusEventTypeMesoChanged    = "MESO_CHANGED"
	StatusEventTypeFameChanged    = "FAME_CHANGED"
	StatusEventTypeDeleted        = "DELETED"
	StatusEventTypeError          = "ERROR"

	StatusEventErrorTypeNotEnoughMeso = "NOT_ENOUGH_MESO"

	EnvEventTopicMovement     = "EVENT_TOPIC_CHARACTER_MOVEMENT"
	MovementTypeNormal        = "NORMAL"
	MovementTypeTeleport      = "TELEPORT"
	MovementTypeStartFallDown = "START_FALL_DOWN"
	MovementTypeFlyingBlock   = "FLYING_BLOCK"
	MovementTypeJump          = "JUMP"
	MovementTypeStatChange    = "STAT_CHANGE"
)

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type statusEventCreatedBody struct {
	Name string `json:"name"`
}

type statusEventLoginBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type statusEventLogoutBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
}

type changeChannelEventLoginBody struct {
	ChannelId    byte   `json:"channelId"`
	OldChannelId byte   `json:"oldChannelId"`
	MapId        uint32 `json:"mapId"`
}

type statusEventMapChangedBody struct {
	ChannelId      byte   `json:"channelId"`
	OldMapId       uint32 `json:"oldMapId"`
	TargetMapId    uint32 `json:"targetMapId"`
	TargetPortalId uint32 `json:"targetPortalId"`
}

type statusEventDeletedBody struct {
}

type statusEventErrorBody[F any] struct {
	Error string `json:"error"`
	Body  F      `json:"body"`
}

type mesoChangedStatusEventBody struct {
	Amount int32 `json:"amount"`
}

type notEnoughMesoErrorStatusBodyBody struct {
	Amount int32 `json:"amount"`
}

type fameChangedStatusEventBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int8   `json:"amount"`
}

type movementEvent struct {
	WorldId     byte     `json:"worldId"`
	ChannelId   byte     `json:"channelId"`
	MapId       uint32   `json:"mapId"`
	CharacterId uint32   `json:"characterId"`
	Movement    Movement `json:"movement"`
}

type Movement struct {
	StartX   int16     `json:"startX"`
	StartY   int16     `json:"startY"`
	Elements []Element `json:"elements"`
}

type Element struct {
	TypeStr     string `json:"typeStr"`
	TypeVal     byte   `json:"typeVal"`
	StartX      int16  `json:"startX"`
	StartY      int16  `json:"startY"`
	MoveAction  byte   `json:"moveAction"`
	Stat        byte   `json:"stat"`
	X           int16  `json:"x"`
	Y           int16  `json:"y"`
	VX          int16  `json:"vX"`
	VY          int16  `json:"vY"`
	FH          int16  `json:"fh"`
	FHFallStart int16  `json:"fhFallStart"`
	XOffset     int16  `json:"xOffset"`
	YOffset     int16  `json:"yOffset"`
	TimeElapsed int16  `json:"timeElapsed"`
}

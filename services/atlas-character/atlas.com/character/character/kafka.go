package character

const (
	EnvEventTopicCharacterStatus       = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeCreated    = "CREATED"
	EventCharacterStatusTypeLogin      = "LOGIN"
	EventCharacterStatusTypeLogout     = "LOGOUT"
	EventCharacterStatusTypeMapChanged = "MAP_CHANGED"

	EnvCommandTopic           = "COMMAND_TOPIC_CHARACTER"
	CommandCharacterChangeMap = "CHANGE_MAP"

	EnvCommandTopicMovement   = "COMMAND_TOPIC_CHARACTER_MOVEMENT"
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

type statusEventMapChangedBody struct {
	ChannelId      byte   `json:"channelId"`
	OldMapId       uint32 `json:"oldMapId"`
	TargetMapId    uint32 `json:"targetMapId"`
	TargetPortalId uint32 `json:"targetPortalId"`
}

type commandEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type changeMapBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
}

type movementCommand struct {
	WorldId     byte     `json:"worldId"`
	ChannelId   byte     `json:"channelId"`
	MapId       uint32   `json:"mapId"`
	CharacterId uint32   `json:"characterId"`
	Movement    movement `json:"movement"`
}

type movementEvent struct {
	WorldId     byte     `json:"worldId"`
	ChannelId   byte     `json:"channelId"`
	MapId       uint32   `json:"mapId"`
	CharacterId uint32   `json:"characterId"`
	Movement    movement `json:"movement"`
}

type movement struct {
	StartX   int16     `json:"startX"`
	StartY   int16     `json:"startY"`
	Elements []element `json:"elements"`
}

type element struct {
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

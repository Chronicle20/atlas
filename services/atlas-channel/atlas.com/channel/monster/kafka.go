package monster

const (
	EnvCommandTopic   = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage = "DAMAGE"
)

type command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MonsterId uint32 `json:"monsterId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type damageCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
}

const (
	EnvCommandMovement = "COMMAND_TOPIC_MONSTER_MOVEMENT"

	MovementTypeNormal        = "NORMAL"
	MovementTypeTeleport      = "TELEPORT"
	MovementTypeStartFallDown = "START_FALL_DOWN"
	MovementTypeFlyingBlock   = "FLYING_BLOCK"
	MovementTypeJump          = "JUMP"
	MovementTypeStatChange    = "STAT_CHANGE"
)

type movementCommand struct {
	WorldId       byte       `json:"worldId"`
	ChannelId     byte       `json:"channelId"`
	UniqueId      uint32     `json:"uniqueId"`
	ObserverId    uint32     `json:"observerId"`
	SkillPossible bool       `json:"skillPossible"`
	Skill         int8       `json:"skill"`
	SkillId       int16      `json:"skillId"`
	SkillLevel    int16      `json:"skillLevel"`
	MultiTarget   []position `json:"multiTarget"`
	RandomTimes   []int32    `json:"randomTimes"`
	Movement      movement   `json:"movement"`
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

type position struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

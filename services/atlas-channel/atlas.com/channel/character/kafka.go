package character

import "atlas-channel/movement"

const (
	EnvCommandTopic            = "COMMAND_TOPIC_CHARACTER"
	CommandRequestDistributeAp = "REQUEST_DISTRIBUTE_AP"
	CommandRequestDistributeSp = "REQUEST_DISTRIBUTE_SP"
	CommandRequestDropMeso     = "REQUEST_DROP_MESO"
	CommandChangeHP            = "CHANGE_HP"
	CommandChangeMP            = "CHANGE_MP"

	CommandDistributeApAbilityStrength     = "STRENGTH"
	CommandDistributeApAbilityDexterity    = "DEXTERITY"
	CommandDistributeApAbilityIntelligence = "INTELLIGENCE"
	CommandDistributeApAbilityLuck         = "LUCK"
	CommandDistributeApAbilityHp           = "HP"
	CommandDistributeApAbilityMp           = "MP"

	EnvCommandTopicMovement = "COMMAND_TOPIC_CHARACTER_MOVEMENT"
)

type command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type DistributePair struct {
	Ability string `json:"ability"`
	Amount  int8   `json:"amount"`
}

type requestDistributeApCommandBody struct {
	Distributions []DistributePair `json:"distributions"`
}

type requestDistributeSpCommandBody struct {
	SkillId uint32 `json:"skilId"`
	Amount  int8   `json:"amount"`
}

type requestDropMesoCommandBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Amount    uint32 `json:"amount"`
}

type changeHPCommandBody struct {
	ChannelId byte  `json:"channelId"`
	Amount    int16 `json:"amount"`
}

type changeMPCommandBody struct {
	ChannelId byte  `json:"channelId"`
	Amount    int16 `json:"amount"`
}

type movementCommand struct {
	WorldId     byte              `json:"worldId"`
	ChannelId   byte              `json:"channelId"`
	MapId       uint32            `json:"mapId"`
	CharacterId uint32            `json:"characterId"`
	Movement    movement.Movement `json:"movement"`
}

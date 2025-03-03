package monster

import "atlas-monsters/monster"

const (
	EnvCommandTopic   = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage = "DAMAGE"

	EnvCommandTopicMovement = "COMMAND_TOPIC_MONSTER_MOVEMENT"
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

type movementCommand struct {
	UniqueId      uint32             `json:"uniqueId"`
	ObserverId    uint32             `json:"observerId"`
	SkillPossible bool               `json:"skillPossible"`
	Skill         int8               `json:"skill"`
	SkillId       int16              `json:"skillId"`
	SkillLevel    int16              `json:"skillLevel"`
	MultiTarget   []monster.Position `json:"multiTarget"`
	RandomTimes   []int32            `json:"randomTimes"`
	Movement      monster.Movement   `json:"movement"`
}

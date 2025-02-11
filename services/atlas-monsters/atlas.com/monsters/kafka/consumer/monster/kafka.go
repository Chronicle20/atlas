package monster

import "atlas-monsters/monster"

const (
	EnvCommandTopicDamage   = "COMMAND_TOPIC_MONSTER_DAMAGE"
	EnvCommandTopicMovement = "COMMAND_TOPIC_MONSTER_MOVEMENT"
)

type damageCommand struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	MapId       uint32 `json:"mapId"`
	UniqueId    uint32 `json:"uniqueId"`
	CharacterId uint32 `json:"characterId"`
	Damage      int64  `json:"damage"`
}

type movementCommand struct {
	WorldId       byte               `json:"worldId"`
	ChannelId     byte               `json:"channelId"`
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

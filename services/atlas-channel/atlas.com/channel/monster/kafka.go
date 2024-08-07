package monster

import "atlas-channel/tenant"

const (
	EnvCommandMovement = "COMMAND_TOPIC_MONSTER_MOVEMENT"
)

type movementCommand struct {
	Tenant        tenant.Model `json:"tenant"`
	WorldId       byte         `json:"worldId"`
	ChannelId     byte         `json:"channelId"`
	UniqueId      uint32       `json:"uniqueId"`
	ObserverId    uint32       `json:"observerId"`
	SkillPossible bool         `json:"skillPossible"`
	Skill         int8         `json:"skill"`
	SkillId       int16        `json:"skillId"`
	SkillLevel    int16        `json:"skillLevel"`
	MultiTarget   []position   `json:"multiTarget"`
	RandomTimes   []int32      `json:"randomTimes"`
	RawMovement   []byte       `json:"rawMovement"`
}

type position struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

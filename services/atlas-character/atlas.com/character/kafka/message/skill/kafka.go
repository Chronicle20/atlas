package skill

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_SKILL"
)

const (
	CommandTypeRequestCreate = "REQUEST_CREATE"
	CommandTypeRequestUpdate = "REQUEST_UPDATE"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type RequestCreateBody struct {
	SkillId     uint32    `json:"skillId"`
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

type RequestUpdateBody struct {
	SkillId     uint32    `json:"skillId"`
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

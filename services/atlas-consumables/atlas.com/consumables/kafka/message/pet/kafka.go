package pet

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_PET"
)

const (
	CommandAwardFullness = "AWARD_FULLNESS"
	CommandSetSkill      = "SET_SKILL"
)

type Command[E any] struct {
	ActorId uint32 `json:"actorId"`
	PetId   uint32 `json:"petId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type AwardFullnessCommandBody struct {
	Amount byte `json:"amount"`
}

// SetSkillCommandBody carries a semantic pet skill key (atlas-constants
// pet/skill spelling) — never a client wire bit.
type SetSkillCommandBody struct {
	Skill   string `json:"skill"`
	Enabled bool   `json:"enabled"`
}

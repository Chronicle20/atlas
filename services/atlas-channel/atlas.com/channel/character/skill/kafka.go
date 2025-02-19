package skill

const (
	EnvCommandTopic        = "COMMAND_TOPIC_SKILL"
	CommandTypeSetCooldown = "SET_COOLDOWN"
)

type command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type setCooldownBody struct {
	SkillId  uint32 `json:"skillId"`
	Cooldown uint32 `json:"cooldown"`
}

package macro

const (
	EnvCommandTopic   = "COMMAND_TOPIC_SKILL_MACRO"
	CommandTypeUpdate = "UPDATE"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type UpdateCommandBody struct {
	Macros []MacroBody `json:"macros"`
}

type MacroBody struct {
	Id       uint32 `json:"id"`
	Name     string `json:"name"`
	Shout    bool   `json:"shout"`
	SkillId1 uint32 `json:"skillId1"`
	SkillId2 uint32 `json:"skillId2"`
	SkillId3 uint32 `json:"skillId3"`
}

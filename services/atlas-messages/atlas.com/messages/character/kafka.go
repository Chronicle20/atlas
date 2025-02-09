package character

const (
	EnvCommandTopic           = "COMMAND_TOPIC_CHARACTER"
	CommandCharacterChangeMap = "CHANGE_MAP"
	CommandChangeJob          = "CHANGE_JOB"
	CommandAwardExperience    = "AWARD_EXPERIENCE"
	CommandAwardLevel         = "AWARD_LEVEL"
)

type command[E any] struct {
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

type changeJobCommandBody struct {
	ChannelId byte   `json:"channelId"`
	JobId     uint16 `json:"jobId"`
}

type awardExperienceCommandBody struct {
	ChannelId byte   `json:"channelId"`
	Amount    uint32 `json:"amount"`
}

type awardLevelCommandBody struct {
	ChannelId byte `json:"channelId"`
	Amount    byte `json:"amount"`
}

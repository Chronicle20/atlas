package pet

const (
	EnvCommandTopic      = "COMMAND_TOPIC_PET"
	CommandAwardFullness = "AWARD_FULLNESS"
)

type command[E any] struct {
	ActorId uint32 `json:"actorId"`
	PetId   uint64 `json:"petId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type awardFullnessCommandBody struct {
	Amount byte `json:"amount"`
}

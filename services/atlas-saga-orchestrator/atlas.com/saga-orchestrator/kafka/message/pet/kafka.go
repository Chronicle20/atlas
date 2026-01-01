package pet

const (
	// EnvCommandTopic defines the environment variable for the pet command topic
	EnvCommandTopic          = "COMMAND_TOPIC_PET"
	// CommandTypeGainCloseness is the command type for gaining closeness with a pet
	CommandTypeGainCloseness = "GAIN_CLOSENESS"
)

type Command[E any] struct {
	PetId uint32 `json:"petId"`
	Type  string `json:"type"`
	Body  E      `json:"body"`
}

// GainClosenessCommandBody represents the body of a gain closeness command.
// This command is used to increase a pet's closeness level.
type GainClosenessCommandBody struct {
	// Amount is the amount of closeness to add to the pet
	Amount uint16 `json:"amount"`
}

package note

const (
	EnvCommandTopic = "COMMAND_TOPIC_NOTE"

	CommandTypeCreate  = "CREATE"
	CommandTypeDiscard = "DISCARD"
)

// Command represents a Kafka command for note operations
type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// CommandCreateBody contains data for creating a note
type CommandCreateBody struct {
	SenderId uint32 `json:"senderId"`
	Message  string `json:"message"`
	Flag     byte   `json:"flag"`
}

// CommandDiscardBody contains data for discarding notes
type CommandDiscardBody struct {
	NoteIds []uint32 `json:"noteIds"`
}

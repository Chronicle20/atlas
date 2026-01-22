package character

// Kafka topic environment variable names
const (
	EnvCommandTopic              = "COMMAND_TOPIC_CHARACTER"
	EnvEventTopicCharacterStatus = "EVENT_TOPIC_CHARACTER_STATUS"
)

// Command types
const (
	CommandChangeMap = "CHANGE_MAP"
)

// Event types
const (
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"
)

package character

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

// Kafka topic environment variable names
const (
	EnvEventTopicCharacterStatus topic.Token = "EVENT_TOPIC_CHARACTER_STATUS"
)

// Event types
const (
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"
)

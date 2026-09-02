package script

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

// Kafka topic environment variable names
const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_PORTAL_ACTIONS"
)

// Command types
const (
	CommandTypeEnter = "ENTER"
)

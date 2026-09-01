package character

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopicCharacterStatus topic.Token = "EVENT_TOPIC_CHARACTER_STATUS"
)

const (
	EventCharacterStatusTypeStatChanged = "STAT_CHANGED"
)

package character

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic          = "COMMAND_TOPIC_CHARACTER"
	CommandRequestChangeFame = "REQUEST_CHANGE_FAME"

	CommandActorTypeCharacter = "CHARACTER"

	EnvEventTopicCharacterStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeDeleted = "DELETED"
)

type CommandEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type RequestChangeFameBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int8   `json:"amount"`
}

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventDeletedBody struct {
}

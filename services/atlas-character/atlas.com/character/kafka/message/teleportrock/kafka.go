package teleportrock

import (
	"github.com/google/uuid"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic  = "COMMAND_TOPIC_TELEPORT_ROCK"
	CommandAddMap    = "ADD_MAP"
	CommandRemoveMap = "REMOVE_MAP"

	EnvEventTopicStatus        = "EVENT_TOPIC_TELEPORT_ROCK_STATUS"
	StatusEventTypeListUpdated = "LIST_UPDATED"
	StatusEventTypeError       = "ERROR"

	ErrorReasonListFull      = "LIST_FULL"
	ErrorReasonDuplicate     = "DUPLICATE"
	ErrorReasonMapNotAllowed = "MAP_NOT_ALLOWED"
	ErrorReasonNotFound      = "NOT_FOUND"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type AddMapCommandBody struct {
	MapId _map.Id `json:"mapId"`
	Vip   bool    `json:"vip"`
}

type RemoveMapCommandBody struct {
	MapId _map.Id `json:"mapId"`
	Vip   bool    `json:"vip"`
}

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// ListUpdatedStatusBody carries the authoritative post-mutation list for the
// affected list only (unpadded). Registered picks REGISTER_LIST vs DELETE_LIST
// on projection (design §4.2).
type ListUpdatedStatusBody struct {
	Vip        bool      `json:"vip"`
	Registered bool      `json:"registered"`
	Maps       []_map.Id `json:"maps"`
}

type ErrorStatusBody struct {
	Vip    bool   `json:"vip"`
	Reason string `json:"reason"`
}

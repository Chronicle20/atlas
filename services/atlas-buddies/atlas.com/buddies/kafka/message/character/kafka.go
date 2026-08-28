package character

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_CHARACTER_STATUS"
)

const (
	StatusEventTypeCreated        = "CREATED"
	StatusEventTypeDeleted        = "DELETED"
	StatusEventTypeLogin          = "LOGIN"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
	StatusEventTypeNameChanged    = "NAME_CHANGED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
	Name string `json:"name"`
}

type DeletedStatusEventBody struct{}

type LoginStatusEventBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type LogoutStatusEventBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type ChannelChangedStatusEventBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
	Instance     uuid.UUID  `json:"instance"`
}

// StatusEventNameChangedBody mirrors the producer's
// atlas-character/kafka/message/character/kafka.go StatusEventNameChangedBody
// field-for-field: OldName/NewName, both `json:"oldName"`/`json:"newName"`.
// There is no compile-time link between the two services, so a wrong tag
// here silently drops every NAME_CHANGED event.
type NameChangedStatusEventBody struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

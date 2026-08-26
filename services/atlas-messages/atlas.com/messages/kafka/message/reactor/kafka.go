package reactor

import (
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

const (
	EnvCommandTopic           = "COMMAND_TOPIC_REACTOR"
	CommandTypeDestroyInField = "DESTROY_IN_FIELD"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type DestroyInFieldCommandBody struct{}

func DestroyInFieldCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := Command[DestroyInFieldCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Instance:  instance,
		Type:      CommandTypeDestroyInField,
		Body:      DestroyInFieldCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

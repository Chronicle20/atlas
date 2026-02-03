package reactor

import (
	reactorKafka "atlas-maps/kafka/message/reactor"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, classification uint32, name string, state int8, x int16, y int16, delay uint32, direction byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &reactorKafka.Command[reactorKafka.CreateCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		MapId:         mapId,
		Instance:      instance,
		Type:          reactorKafka.CommandTypeCreate,
		Body: reactorKafka.CreateCommandBody{
			Classification: classification,
			Name:           name,
			State:          state,
			X:              x,
			Y:              y,
			Delay:          delay,
			Direction:      direction,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

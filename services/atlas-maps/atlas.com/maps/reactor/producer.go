package reactor

import (
	reactorKafka "atlas-maps/kafka/message/reactor"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(transactionId uuid.UUID, field field.Model, classification uint32, name string, state int8, x int16, y int16, delay uint32, direction byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(field.MapId()))
	value := &reactorKafka.Command[reactorKafka.CreateCommandBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		ChannelId:     field.ChannelId(),
		MapId:         field.MapId(),
		Instance:      field.Instance(),
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

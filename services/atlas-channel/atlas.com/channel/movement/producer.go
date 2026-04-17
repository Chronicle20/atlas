package movement

import (
	"atlas-channel/kafka/message/movement"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CommandProducer(f field.Model, objectId uint64, observerId uint32, x int16, y int16, stance byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(objectId))

	value := &movement.Command[any]{
		WorldId:    f.WorldId(),
		ChannelId:  f.ChannelId(),
		MapId:      f.MapId(),
		Instance:   f.Instance(),
		ObjectId:   objectId,
		ObserverId: observerId,
		X:          x,
		Y:          y,
		Stance:     stance,
	}
	return producer.SingleMessageProvider(key, value)
}

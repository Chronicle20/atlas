package movement

import (
	"atlas-channel/kafka/message/movement"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CommandProducer(m _map.Model, objectId uint64, observerId uint32, x int16, y int16, stance byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(objectId))

	value := &movement.Command[any]{
		WorldId:    world.Id(m.WorldId()),
		ChannelId:  channel.Id(m.ChannelId()),
		MapId:      _map.Id(m.MapId()),
		ObjectId:   objectId,
		ObserverId: observerId,
		X:          x,
		Y:          y,
		Stance:     stance,
	}
	return producer.SingleMessageProvider(key, value)
}

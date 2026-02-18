package drop

import (
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func SpawnDropCommandProvider(f field.Model, itemId uint32, quantity uint32, mesos uint32, x int16, y int16, dropperId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	cmd := newSpawnCommand(f, spawnCommandBody{
		ItemId:       itemId,
		Quantity:     quantity,
		Mesos:        mesos,
		DropType:     0,
		X:            x,
		Y:            y,
		OwnerId:      0,
		OwnerPartyId: 0,
		DropperId:    dropperId,
		DropperX:     x,
		DropperY:     y,
		PlayerDrop:   false,
		Mod:          1,
	})
	return producer.SingleMessageProvider(key, &cmd)
}

package pet

import (
	"atlas-channel/movement"
	model2 "atlas-channel/socket/model"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func Move(petId uint64, ma _map.Model, characterId uint32, mm model2.Movement) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))

	value := &movementCommand{
		WorldId:     byte(ma.WorldId()),
		ChannelId:   byte(ma.ChannelId()),
		MapId:       uint32(ma.MapId()),
		PetId:       petId,
		CharacterId: characterId,
		Movement:    movement.ProduceMovementForKafka(mm),
	}
	return producer.SingleMessageProvider(key, value)
}

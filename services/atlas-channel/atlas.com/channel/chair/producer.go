package chair

import (
	"atlas-channel/kafka/message/chair"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func UseCommandProvider(m _map.Model, chairType string, chairId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chair.Command[chair.UseChairCommandBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		Type:      chair.CommandUseChair,
		Body: chair.UseChairCommandBody{
			CharacterId: characterId,
			ChairType:   chairType,
			ChairId:     chairId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelCommandProvider(m _map.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chair.Command[chair.CancelChairCommandBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		Type:      chair.CommandCancelChair,
		Body: chair.CancelChairCommandBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

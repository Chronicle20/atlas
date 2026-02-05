package chair

import (
	"atlas-channel/kafka/message/chair"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func UseCommandProvider(f field.Model, chairType string, chairId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chair.Command[chair.UseChairCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      chair.CommandUseChair,
		Body: chair.UseChairCommandBody{
			CharacterId: characterId,
			ChairType:   chairType,
			ChairId:     chairId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chair.Command[chair.CancelChairCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      chair.CommandCancelChair,
		Body: chair.CancelChairCommandBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

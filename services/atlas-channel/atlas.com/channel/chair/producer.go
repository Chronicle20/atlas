package chair

import (
	"atlas-channel/kafka/message/chair"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func RecoveryCommandProvider(f field.Model, characterId uint32, hp int16, mp int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chair.Command[chair.RecoveryCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      chair.CommandRecovery,
		Body: chair.RecoveryCommandBody{
			CharacterId: characterId,
			Hp:          hp,
			Mp:          mp,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

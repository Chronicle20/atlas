package _map

import (
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/message/mapactions"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func enterMapProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.CharacterEnter]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeCharacterEnter,
		Body: mapKafka.CharacterEnter{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func exitMapProvider(transactionId uuid.UUID, f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.CharacterExit]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeCharacterExit,
		Body: mapKafka.CharacterExit{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func enterMapActionsProvider(transactionId uuid.UUID, f field.Model, characterId uint32, scriptName string, scriptType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapactions.Command[mapactions.EnterCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapactions.CommandTypeEnter,
		Body: mapactions.EnterCommandBody{
			CharacterId: characterId,
			ScriptName:  scriptName,
			ScriptType:  scriptType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

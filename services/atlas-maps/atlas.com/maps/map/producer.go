package _map

import (
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/message/mapactions"
	monsterKafka "atlas-maps/kafka/message/monster"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// destroyFieldCommandProvider despawns every monster in a field. Emitted only
// when a field that actually fired a one-time batch empties (design D7): without
// it, "party kills 6 of 10 Lucidas, leaves, next party enters" yields 4
// survivors plus 10 fresh monsters. Gating on the fired flag keeps the 4,207
// unaffected maps bit-identical to main.
func destroyFieldCommandProvider(f field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &monsterKafka.FieldCommand[monsterKafka.DestroyFieldBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      monsterKafka.CommandTypeDestroyField,
		Body:      monsterKafka.DestroyFieldBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

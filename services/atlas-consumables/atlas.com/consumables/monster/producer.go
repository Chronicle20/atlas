package monster

import (
	monsterMsg "atlas-consumables/kafka/message/monster"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// catchCommandProvider keys on the monster's unique id so concurrent catch
// attempts on one mob land on a single partition and are resolved in order.
func catchCommandProvider(f field.Model, monsterUniqueId uint32, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterUniqueId))
	value := &monsterMsg.Command[monsterMsg.CatchCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterUniqueId,
		Type:      monsterMsg.CommandTypeCatch,
		Body: monsterMsg.CatchCommandBody{
			CharacterId: characterId,
			ItemId:      itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

package consumable

import (
	"atlas-saga-orchestrator/kafka/message/consumable"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// ApplyConsumableEffectCommandProvider creates a Kafka message for applying consumable effects without consuming
func ApplyConsumableEffectCommandProvider(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, characterId character.Id, itemId item.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.ApplyConsumableEffectBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          consumable.CommandApplyConsumableEffect,
		Body: consumable.ApplyConsumableEffectBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

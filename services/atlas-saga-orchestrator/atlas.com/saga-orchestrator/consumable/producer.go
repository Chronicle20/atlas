package consumable

import (
	"atlas-saga-orchestrator/kafka/message/consumable"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// ApplyConsumableEffectCommandProvider creates a Kafka message for applying consumable effects without consuming
func ApplyConsumableEffectCommandProvider(worldId byte, channelId byte, characterId uint32, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.ApplyConsumableEffectBody]{
		WorldId:     worldId,
		ChannelId:   channelId,
		CharacterId: characterId,
		Type:        consumable.CommandApplyConsumableEffect,
		Body: consumable.ApplyConsumableEffectBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

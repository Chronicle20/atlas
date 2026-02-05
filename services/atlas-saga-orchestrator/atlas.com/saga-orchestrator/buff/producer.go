package buff

import (
	buffMsg "atlas-saga-orchestrator/kafka/message/buff"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// CancelAllCommandProvider creates a Kafka message for canceling all buffs on a character
func CancelAllCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buffMsg.Command[buffMsg.CancelAllBody]{
		WorldId:     worldId,
		ChannelId:   channelId,
		MapId:       mapId,
		Instance:    instance,
		CharacterId: characterId,
		Type:        buffMsg.CommandTypeCancelAll,
		Body:        buffMsg.CancelAllBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

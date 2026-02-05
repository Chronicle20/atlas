package portal

import (
	portalMsg "atlas-saga-orchestrator/kafka/message/portal"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// BlockCommandProvider creates a Kafka message for blocking a portal for a character
func BlockCommandProvider(characterId uint32, mapId _map.Id, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &portalMsg.Command[portalMsg.BlockBody]{
		WorldId:   0, // Not needed for blocking
		ChannelId: 0, // Not needed for blocking
		MapId:     mapId,
		PortalId:  portalId,
		Type:      portalMsg.CommandTypeBlock,
		Body: portalMsg.BlockBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// UnblockCommandProvider creates a Kafka message for unblocking a portal for a character
func UnblockCommandProvider(characterId uint32, mapId _map.Id, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &portalMsg.Command[portalMsg.UnblockBody]{
		WorldId:   0, // Not needed for unblocking
		ChannelId: 0, // Not needed for unblocking
		MapId:     mapId,
		PortalId:  portalId,
		Type:      portalMsg.CommandTypeUnblock,
		Body: portalMsg.UnblockBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

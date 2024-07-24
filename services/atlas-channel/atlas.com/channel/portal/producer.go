package portal

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func enterCommandProvider(tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	return func(worldId byte, channelId byte, mapId uint32, portalId uint32, characterId uint32) model.Provider[[]kafka.Message] {
		key := producer.CreateKey(int(portalId))
		value := commandEvent[enterBody]{
			Tenant:    tenant,
			WorldId:   worldId,
			ChannelId: channelId,
			MapId:     mapId,
			PortalId:  portalId,
			Type:      CommandTypeEnter,
			Body: enterBody{
				CharacterId: characterId,
			},
		}
		return producer.SingleMessageProvider(key, value)
	}
}

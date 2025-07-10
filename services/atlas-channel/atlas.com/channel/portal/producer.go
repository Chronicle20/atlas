package portal

import (
	portal2 "atlas-channel/kafka/message/portal"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func EnterCommandProvider(m _map.Model, portalId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(portalId))
	value := portal2.Command[portal2.EnterBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		PortalId:  portalId,
		Type:      portal2.CommandTypeEnter,
		Body: portal2.EnterBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

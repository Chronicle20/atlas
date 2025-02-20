package portal

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func enterCommandProvider(m _map.Model, portalId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(portalId))
	value := commandEvent[enterBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		PortalId:  portalId,
		Type:      CommandTypeEnter,
		Body: enterBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

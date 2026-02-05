package portal

import (
	portal2 "atlas-channel/kafka/message/portal"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func EnterCommandProvider(f field.Model, portalId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(portalId))
	value := portal2.Command[portal2.EnterBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		PortalId:  portalId,
		Type:      portal2.CommandTypeEnter,
		Body: portal2.EnterBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

package portal

import (
	portal2 "atlas-channel/kafka/message/portal"

	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
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

func WarpCommandProvider(f field.Model, characterId uint32, targetMapId _map.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := portal2.WarpCommand{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      portal2.CommandTypeWarp,
		Body: portal2.WarpBody{
			CharacterId: characterId,
			TargetMapId: targetMapId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

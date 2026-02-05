package character

import (
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func enableActionsProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventStatChangedBody]{
		CharacterId: characterId,
		Type:        EventCharacterStatusTypeStatChanged,
		WorldId:     f.WorldId(),
		Body: statusEventStatChangedBody{
			ChannelId:       f.ChannelId(),
			ExclRequestSent: true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeMapProvider(f field.Model, characterId uint32, targetMapId _map.Id, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[changeMapBody]{
		WorldId:     f.WorldId(),
		CharacterId: characterId,
		Type:        CommandCharacterChangeMap,
		Body: changeMapBody{
			ChannelId: f.ChannelId(),
			MapId:     targetMapId,
			Instance:  f.Instance(),
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

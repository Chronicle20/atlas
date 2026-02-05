package character

import (
	character2 "atlas-transports/kafka/message/character"

	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func ChangeMapProvider(f field.Model, characterId uint32, targetMapId _map.Id, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMapBody]{
		WorldId:     f.WorldId(),
		CharacterId: characterId,
		Type:        character2.CommandCharacterChangeMap,
		Body: character2.ChangeMapBody{
			ChannelId: f.ChannelId(),
			MapId:     targetMapId,
			Instance:  f.Instance(),
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

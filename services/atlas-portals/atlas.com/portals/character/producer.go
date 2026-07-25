package character

import (
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// ChangeToPositionProvider issues a CHANGE_MAP command that lands the character
// at an exact (x, y) coordinate in the target map rather than a named portal —
// used by Mystic Door to place the user on the linked door's position.
func ChangeToPositionProvider(f field.Model, characterId uint32, targetMapId _map.Id, x int16, y int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[changeMapBody]{
		WorldId:     f.WorldId(),
		CharacterId: characterId,
		Type:        CommandCharacterChangeMap,
		Body: changeMapBody{
			ChannelId:         f.ChannelId(),
			MapId:             targetMapId,
			Instance:          f.Instance(),
			UseTargetPosition: true,
			TargetX:           x,
			TargetY:           y,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

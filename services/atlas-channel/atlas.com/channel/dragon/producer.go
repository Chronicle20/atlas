package dragon

import (
	dragonmsg "atlas-channel/kafka/message/dragon"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func CreateCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &dragonmsg.Command[dragonmsg.CreateCommandBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance(),
		Type: dragonmsg.CommandTypeCreate,
		Body: dragonmsg.CreateCommandBody{CharacterId: characterId},
	}
	return producer.SingleMessageProvider(key, value)
}

func DestroyCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &dragonmsg.Command[dragonmsg.DestroyCommandBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance(),
		Type: dragonmsg.CommandTypeDestroy,
		Body: dragonmsg.DestroyCommandBody{CharacterId: characterId},
	}
	return producer.SingleMessageProvider(key, value)
}

// MoveCommandProvider keys on the OWNER character id, not the map: dragon moves
// for one character must stay ordered relative to each other.
func MoveCommandProvider(f field.Model, characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &dragonmsg.Command[dragonmsg.MoveCommandBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance(),
		Type: dragonmsg.CommandTypeMove,
		Body: dragonmsg.MoveCommandBody{
			CharacterId: characterId, StartX: startX, StartY: startY, Stance: stance, RawMovement: rawMovement,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

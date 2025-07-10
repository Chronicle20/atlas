package drop

import (
	drop2 "atlas-channel/kafka/message/drop"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestReservationCommandProvider(m _map.Model, dropId uint32, characterId uint32, characterX int16, characterY int16, petSlot int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &drop2.Command[drop2.RequestReservationCommandBody]{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		Type:      drop2.CommandTypeRequestReservation,
		Body: drop2.RequestReservationCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
			CharacterX:  characterX,
			CharacterY:  characterY,
			PetSlot:     petSlot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

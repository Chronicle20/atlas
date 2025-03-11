package drop

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestReservationCommandProvider(m _map.Model, dropId uint32, characterId uint32, characterX int16, characterY int16, petSlot int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &command[requestReservationCommandBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		Type:      CommandTypeRequestReservation,
		Body: requestReservationCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
			CharacterX:  characterX,
			CharacterY:  characterY,
			PetSlot:     petSlot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

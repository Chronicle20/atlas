package drop

import (
	drop2 "atlas-channel/kafka/message/drop"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestReservationCommandProvider(f field.Model, dropId uint32, characterId uint32, partyId uint32, characterX int16, characterY int16, petSlot int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &drop2.Command[drop2.RequestReservationCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      drop2.CommandTypeRequestReservation,
		Body: drop2.RequestReservationCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
			PartyId:     partyId,
			CharacterX:  characterX,
			CharacterY:  characterY,
			PetSlot:     petSlot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

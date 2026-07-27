package drop

import (
	drop2 "atlas-channel/kafka/message/drop"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// SpawnMesoCommandProvider emits a meso-only FFA drop: ItemId=0,
// Quantity=0, DropType=2 (client-visual FFA styling; the server ignores
// it), PlayerDrop=true (universal pickup via CanBeReservedBy),
// OwnerPartyId=0, Mod=0 (no client animation delay; atlas-drops discards
// the field today).
func SpawnMesoCommandProvider(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropperId))
	value := &drop2.Command[drop2.SpawnCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      drop2.CommandTypeSpawn,
		Body: drop2.SpawnCommandBody{
			ItemId:       0,
			Quantity:     0,
			Mesos:        mesos,
			DropType:     2,
			X:            x,
			Y:            y,
			OwnerId:      ownerId,
			OwnerPartyId: 0,
			DropperId:    dropperId,
			DropperX:     dropperX,
			DropperY:     dropperY,
			PlayerDrop:   true,
			Mod:          0,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

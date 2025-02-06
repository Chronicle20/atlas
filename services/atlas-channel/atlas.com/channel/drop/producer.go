package drop

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestReservationCommandProvider(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, characterX int16, characterY int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &command[requestReservationCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      CommandTypeRequestReservation,
		Body: requestReservationCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
			CharacterX:  characterX,
			CharacterY:  characterY,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

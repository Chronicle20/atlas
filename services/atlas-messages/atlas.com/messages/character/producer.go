package character

import (
	character2 "atlas-messages/kafka/message/character"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)


func changeJobCommandProvider(characterId uint32, worldId byte, channelId byte, jobId uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeJobCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        character2.CommandChangeJob,
		Body: character2.ChangeJobCommandBody{
			ChannelId: channelId,
			JobId:     jobId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}


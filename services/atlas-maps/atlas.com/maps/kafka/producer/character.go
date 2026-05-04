package producer

import (
	characterKafka "atlas-maps/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// ChannelChangedStatusProvider builds a CHANNEL_CHANGED status event for the
// EVENT_TOPIC_CHARACTER_STATUS topic. atlas-maps emits this when it resolves
// a CHANNEL_CHANGE_REQUEST so downstream consumers (and atlas-channel) see the
// canonical post-resolution channel/map.
func ChannelChangedStatusProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, oldChannelId channel.Id, newField field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          characterKafka.EventCharacterStatusTypeChannelChanged,
		Body: characterKafka.ChangeChannelEventLoginBody{
			ChannelId:    newField.ChannelId(),
			OldChannelId: oldChannelId,
			MapId:        newField.MapId(),
			Instance:     newField.Instance(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

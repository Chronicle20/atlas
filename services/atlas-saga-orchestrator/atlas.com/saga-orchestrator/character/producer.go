package character

import (
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func ChangeMapProvider(transactionId uuid.UUID, characterId uint32, field field.Model, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMapBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandChangeMap,
		Body: character2.ChangeMapBody{
			ChannelId: field.ChannelId(),
			MapId:     field.MapId(),
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

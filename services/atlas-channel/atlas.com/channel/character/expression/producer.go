package expression

import (
	expression2 "atlas-channel/kafka/message/expression"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func SetCommandProvider(characterId uint32, f field.Model, expression uint32, duration int32, byItemOption bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	// atlas-expressions' Command has always declared transactionId but this
	// producer never set it, so every command arrived with the zero UUID and
	// carried it onto every StatusEvent. One id per command emitted.
	value := &expression2.Command{
		TransactionId: uuid.New(),
		CharacterId:   characterId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Expression:    expression,
		Duration:      duration,
		ByItemOption:  byItemOption,
	}
	return producer.SingleMessageProvider(key, value)
}

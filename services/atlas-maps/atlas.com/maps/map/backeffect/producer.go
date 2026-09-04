package backeffect

import (
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func BackEffectSetEventProvider(transactionId uuid.UUID, f field.Model, e BackEffectEntry) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.BackEffectSet]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeBackEffectSet,
		Body: mapKafka.BackEffectSet{
			Effect:   e.Effect,
			FieldId:  e.FieldId,
			PageId:   e.PageId,
			Duration: e.Duration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func BackEffectClearEventProvider(transactionId uuid.UUID, f field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.BackEffectClear]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeBackEffectClear,
		Body:          mapKafka.BackEffectClear{},
	}
	return producer.SingleMessageProvider(key, value)
}

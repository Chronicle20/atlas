package environment

import (
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func EnvironmentStateChangedEventProvider(transactionId uuid.UUID, f field.Model, e ObjectEntry) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.EnvironmentStateChanged]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeEnvironmentStateChanged,
		Body: mapKafka.EnvironmentStateChanged{
			Kind:  string(e.Kind),
			Name:  e.Name,
			State: e.State,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func EnvironmentResetEventProvider(transactionId uuid.UUID, f field.Model, cleared []ObjectEntry) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	objects := make([]mapKafka.EnvironmentObject, 0, len(cleared))
	for _, e := range cleared {
		objects = append(objects, mapKafka.EnvironmentObject{Kind: string(e.Kind), Name: e.Name, State: e.DefaultState})
	}
	value := &mapKafka.StatusEvent[mapKafka.EnvironmentReset]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeEnvironmentReset,
		Body:          mapKafka.EnvironmentReset{Cleared: objects},
	}
	return producer.SingleMessageProvider(key, value)
}

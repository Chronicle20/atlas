package jukebox

import (
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func JukeboxStartEventProvider(transactionId uuid.UUID, f field.Model, itemId uint32, playerName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.JukeboxStart]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeJukeboxStart,
		Body: mapKafka.JukeboxStart{
			ItemId:     itemId,
			PlayerName: playerName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func JukeboxEndEventProvider(transactionId uuid.UUID, f field.Model, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.JukeboxEnd]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeJukeboxEnd,
		Body: mapKafka.JukeboxEnd{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

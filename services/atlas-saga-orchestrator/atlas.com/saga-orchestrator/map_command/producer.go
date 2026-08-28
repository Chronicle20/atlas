package map_command

import (
	mapKafka "atlas-saga-orchestrator/kafka/message/map"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/backeffect"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func WeatherStartCommandProvider(transactionId uuid.UUID, f field.Model, itemId uint32, message string, durationMs uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.WeatherStartCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeWeatherStart,
		Body: mapKafka.WeatherStartCommandBody{
			ItemId:     itemId,
			Message:    message,
			DurationMs: durationMs,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func PlayJukeboxCommandProvider(transactionId uuid.UUID, f field.Model, itemId uint32, playerName string, durationMs uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.PlayJukeboxCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypePlayJukebox,
		Body: mapKafka.PlayJukeboxCommandBody{
			ItemId:     itemId,
			PlayerName: playerName,
			DurationMs: durationMs,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func SetEnvironmentStateCommandProvider(transactionId uuid.UUID, f field.Model, kind field.ObjectKind, name string, state uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeSetEnvironmentState,
		Body: mapKafka.SetEnvironmentStateCommandBody{
			Kind:  string(kind),
			Name:  name,
			State: state,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ResetEnvironmentCommandProvider(transactionId uuid.UUID, f field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.ResetEnvironmentCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeResetEnvironment,
		Body:          mapKafka.ResetEnvironmentCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func SetBackEffectCommandProvider(transactionId uuid.UUID, f field.Model, effect backeffect.Effect, fieldId uint32, pageId uint8, duration uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.SetBackEffectCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeSetBackEffect,
		Body: mapKafka.SetBackEffectCommandBody{
			Effect:   effect,
			FieldId:  fieldId,
			PageId:   pageId,
			Duration: duration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ClearBackEffectCommandProvider(transactionId uuid.UUID, f field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.ClearBackEffectCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeClearBackEffect,
		Body:          mapKafka.ClearBackEffectCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

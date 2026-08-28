package _map

import (
	consumer2 "atlas-maps/kafka/consumer"
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/map/backeffect"
	"atlas-maps/map/environment"
	"atlas-maps/map/jukebox"
	"atlas-maps/map/weather"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_command")(mapKafka.EnvCommandTopicMap)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		var err error
		t, err = topic.EnvProvider(l)(mapKafka.EnvCommandTopicMap)()
		if err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleWeatherStartCommand()))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handlePlayJukeboxCommand()))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSetEnvironmentStateCommand()))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleResetEnvironmentCommand()))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSetBackEffectCommand()))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleClearBackEffectCommand()))); err != nil {
			return err
		}
		return nil
	}
}

func handleWeatherStartCommand() func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.WeatherStartCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.WeatherStartCommandBody]) {
		if c.Type != mapKafka.CommandTypeWeatherStart {
			return
		}

		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
		duration := time.Duration(c.Body.DurationMs) * time.Millisecond

		const maxWeatherDuration = 20 * time.Second
		if duration > maxWeatherDuration {
			l.Warnf("Weather duration [%s] for map [%d] instance [%s] exceeds maximum, capping at [%s].", duration, c.MapId, c.Instance, maxWeatherDuration)
			duration = maxWeatherDuration
		}

		l.Debugf("Received weather start command for map [%d] instance [%s] item [%d] duration [%s].", c.MapId, c.Instance, c.Body.ItemId, duration)

		weather.NewProcessor(l, ctx).Start(f, c.Body.ItemId, c.Body.Message, duration)

		err := producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(weather.WeatherStartEventProvider(c.TransactionId, f, c.Body.ItemId, c.Body.Message))
		if err != nil {
			l.WithError(err).Errorf("Unable to produce weather start event for map [%d] instance [%s].", c.MapId, c.Instance)
		}
	}
}

// maxJukeboxDuration bounds a crafted or buggy PLAY_JUKEBOX command. The
// duration is the client's own IWzSound::length, so a real track is well
// under this; ten minutes is an order of magnitude above any real WZ sound
// while still preventing a field's BGM from being pinned indefinitely.
const maxJukeboxDuration = 10 * time.Minute

func handlePlayJukeboxCommand() func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.PlayJukeboxCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.PlayJukeboxCommandBody]) {
		if c.Type != mapKafka.CommandTypePlayJukebox {
			return
		}

		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
		duration := time.Duration(c.Body.DurationMs) * time.Millisecond

		if duration > maxJukeboxDuration {
			l.Warnf("Jukebox duration [%s] for map [%d] instance [%s] exceeds maximum, capping at [%s].", duration, c.MapId, c.Instance, maxJukeboxDuration)
			duration = maxJukeboxDuration
		}

		l.Debugf("Received play jukebox command for map [%d] instance [%s] item [%d] duration [%s].", c.MapId, c.Instance, c.Body.ItemId, duration)

		jukebox.NewProcessor(l, ctx).Start(f, c.Body.ItemId, c.Body.PlayerName, duration)

		err := producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(jukebox.JukeboxStartEventProvider(c.TransactionId, f, c.Body.ItemId, c.Body.PlayerName))
		if err != nil {
			l.WithError(err).Errorf("Unable to produce jukebox start event for map [%d] instance [%s].", c.MapId, c.Instance)
		}
	}
}

func handleSetEnvironmentStateCommand() func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]) {
		if c.Type != mapKafka.CommandTypeSetEnvironmentState {
			return
		}

		kind, err := field.ParseObjectKind(c.Body.Kind)
		if err != nil {
			l.WithError(err).Errorf("Rejecting environment state command for map [%d] instance [%s].", c.MapId, c.Instance)
			return
		}

		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
		entry, err := environment.NewProcessor(l, ctx).Set(f, kind, c.Body.Name, c.Body.State)
		if err != nil {
			l.WithError(err).Errorf("Rejecting environment state command for map [%d] instance [%s].", c.MapId, c.Instance)
			return
		}

		// Emitted unconditionally, including for a re-set to the same state:
		// scripts may rely on the re-broadcast to re-run the client animation.
		err = producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(environment.EnvironmentStateChangedEventProvider(c.TransactionId, f, entry))
		if err != nil {
			l.WithError(err).Errorf("Unable to produce environment state changed event for map [%d] instance [%s].", c.MapId, c.Instance)
		}
	}
}

func handleResetEnvironmentCommand() func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.ResetEnvironmentCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.ResetEnvironmentCommandBody]) {
		if c.Type != mapKafka.CommandTypeResetEnvironment {
			return
		}

		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
		cleared := environment.NewProcessor(l, ctx).Reset(f)

		err := producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(environment.EnvironmentResetEventProvider(c.TransactionId, f, cleared))
		if err != nil {
			l.WithError(err).Errorf("Unable to produce environment reset event for map [%d] instance [%s].", c.MapId, c.Instance)
		}
	}
}

func handleSetBackEffectCommand() func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.SetBackEffectCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.SetBackEffectCommandBody]) {
		if c.Type != mapKafka.CommandTypeSetBackEffect {
			return
		}

		if c.Body.Effect != 0 && c.Body.Effect != 1 {
			l.Warnf("Rejecting set back effect command with invalid effect [%d] for map [%d] instance [%s].", c.Body.Effect, c.MapId, c.Instance)
			return
		}

		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
		entry := backeffect.BackEffectEntry{
			Effect:  c.Body.Effect,
			FieldId: c.Body.FieldId,
			PageId:  c.Body.PageId,
			// Duration is not clamped: it is a fade length bounded by the
			// client's own tween, with no denial-of-service shape comparable
			// to pinning a field's BGM (the counterpart to maxJukeboxDuration
			// above).
			Duration: c.Body.Duration,
		}

		l.Debugf("Received set back effect command for map [%d] instance [%s] page [%d] effect [%d] duration [%d].", c.MapId, c.Instance, c.Body.PageId, c.Body.Effect, c.Body.Duration)

		backeffect.NewProcessor(l, ctx).Set(f, entry)

		err := producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(backeffect.BackEffectSetEventProvider(c.TransactionId, f, entry))
		if err != nil {
			l.WithError(err).Errorf("Unable to produce back effect set event for map [%d] instance [%s].", c.MapId, c.Instance)
		}
	}
}

func handleClearBackEffectCommand() func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.ClearBackEffectCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.ClearBackEffectCommandBody]) {
		if c.Type != mapKafka.CommandTypeClearBackEffect {
			return
		}

		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()

		if !backeffect.NewProcessor(l, ctx).Clear(f) {
			l.Debugf("Received clear back effect command for map [%d] instance [%s] with no active entries.", c.MapId, c.Instance)
		}

		err := producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(backeffect.BackEffectClearEventProvider(c.TransactionId, f))
		if err != nil {
			l.WithError(err).Errorf("Unable to produce back effect clear event for map [%d] instance [%s].", c.MapId, c.Instance)
		}
	}
}

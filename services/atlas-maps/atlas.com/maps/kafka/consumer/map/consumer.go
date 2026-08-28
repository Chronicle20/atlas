package _map

import (
	consumer2 "atlas-maps/kafka/consumer"
	mapKafka "atlas-maps/kafka/message/map"
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
		t, _ = topic.EnvProvider(l)(mapKafka.EnvCommandTopicMap)()
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

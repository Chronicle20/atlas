package _map

import (
	consumer2 "atlas-maps/kafka/consumer"
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/producer"
	"atlas-maps/map/weather"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_command")(mapKafka.EnvCommandTopicMap)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
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

package _map

import (
	"atlas-messages/character"
	"atlas-messages/command"
	mapKafka "atlas-messages/kafka/message/map"
	"context"
	"regexp"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func WeatherCommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(_ context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`@weather\s+(\d+)\s+(.+)`)
			match := re.FindStringSubmatch(m)
			if len(match) != 3 {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			itemId, err := strconv.ParseUint(match[1], 10, 32)
			if err != nil {
				return nil, false
			}

			message := match[2]

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					return producer.ProviderImpl(l)(ctx)(mapKafka.EnvCommandTopicMap)(weatherStartCommandProvider(f, uint32(itemId), message))
				}
			}, true
		}
	}
}

func weatherStartCommandProvider(f field.Model, itemId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.WeatherStartCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeWeatherStart,
		Body: mapKafka.WeatherStartCommandBody{
			ItemId:     itemId,
			Message:    message,
			DurationMs: 30000,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

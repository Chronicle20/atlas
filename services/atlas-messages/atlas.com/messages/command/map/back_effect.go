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

func BackEffectCommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(_ context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`^@backeffect\s+(\d+)\s+([01])(?:\s+(\d+))?$`)
			match := re.FindStringSubmatch(m)
			if len(match) != 4 {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			pageId, err := strconv.ParseUint(match[1], 10, 8)
			if err != nil {
				return nil, false
			}

			effect, err := strconv.ParseUint(match[2], 10, 8)
			if err != nil {
				return nil, false
			}

			var durationMs uint64
			if match[3] != "" {
				durationMs, err = strconv.ParseUint(match[3], 10, 32)
				if err != nil {
					return nil, false
				}
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					return producer.ProviderImpl(l)(ctx)(mapKafka.EnvCommandTopicMap)(setBackEffectCommandProvider(f, uint8(pageId), uint8(effect), uint32(durationMs)))
				}
			}, true
		}
	}
}

func setBackEffectCommandProvider(f field.Model, pageId uint8, effect uint8, durationMs uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.SetBackEffectCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeSetBackEffect,
		Body: mapKafka.SetBackEffectCommandBody{
			Effect:   effect,
			FieldId:  uint32(f.MapId()),
			PageId:   pageId,
			Duration: durationMs,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ClearBackEffectCommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(_ context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			re := regexp.MustCompile(`^@clearbackeffect$`)
			match := re.FindStringSubmatch(m)
			if len(match) != 1 {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					return producer.ProviderImpl(l)(ctx)(mapKafka.EnvCommandTopicMap)(clearBackEffectCommandProvider(f))
				}
			}, true
		}
	}
}

func clearBackEffectCommandProvider(f field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.ClearBackEffectCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeClearBackEffect,
		Body:          mapKafka.ClearBackEffectCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

// Package monsterstatus wires the atlas-monsters EVENT_TOPIC_MONSTER_STATUS
// events into the CRIMSON_BALROG monster processor. Per FR-N18, this package
// is consumer plumbing only — decode, delegate. All elimination-tracking
// logic lives in events/crimsonbalrog (monsters.go).
package monsterstatus

import (
	"atlas-events/events/crimsonbalrog"
	consumer2 "atlas-events/kafka/consumer"
	monsterstatus2 "atlas-events/kafka/message/monsterstatus"
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_status_event")(monsterstatus2.EnvEventTopicMonsterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(monsterstatus2.EnvEventTopicMonsterStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMonsterStatus(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleMonsterStatus(db *gorm.DB) message.Handler[monsterstatus2.StatusEvent[json.RawMessage]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monsterstatus2.StatusEvent[json.RawMessage]) {
		if err := crimsonbalrog.NewMonsterProcessor(l, ctx, db).OnMonsterStatus(e); err != nil {
			l.WithError(err).Errorf("Unable to process monster status [%s] for unique id [%d].", e.Type, e.UniqueId)
		}
	}
}

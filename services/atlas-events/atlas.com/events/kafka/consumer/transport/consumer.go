// Package transport wires the atlas-transports VOYAGE_DEPARTED status event
// into the CRIMSON_BALROG trigger processor. Per FR-N18, this package is
// consumer plumbing only — decode, type-guard, delegate. All scheduling
// logic lives in events/crimsonbalrog.
package transport

import (
	"atlas-events/events/crimsonbalrog"
	consumer2 "atlas-events/kafka/consumer"
	transport2 "atlas-events/kafka/message/transport"
	"context"

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
			rf(consumer2.NewConfig(l)("transport_status_event")(transport2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(transport2.EnvEventTopicStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleVoyageDeparted(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleVoyageDeparted(db *gorm.DB) message.Handler[transport2.StatusEvent[transport2.VoyageStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e transport2.StatusEvent[transport2.VoyageStatusEventBody]) {
		if e.Type != transport2.EventStatusVoyageDeparted {
			return
		}
		if err := crimsonbalrog.NewTriggerProcessor(l, ctx, db).OnVoyageDeparted(e); err != nil {
			l.WithError(err).Errorf("Unable to schedule Crimson Balrog trigger evaluation for voyage [%s].", e.Body.VoyageId)
		}
	}
}

package equipable

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	consumer2 "atlas-inventory/kafka/consumer"
	equipable2 "atlas-inventory/kafka/message/equipable"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("equipable_status_event")(equipable2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(equipable2.EnvStatusEventTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdatedStatusEvent(db))))
		}
	}
}

func handleUpdatedStatusEvent(db *gorm.DB) message.Handler[equipable2.StatusEvent[equipable2.AttributeBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e equipable2.StatusEvent[equipable2.AttributeBody]) {
		if e.Type != equipable2.StatusEventUpdated {
			return
		}

		ap := asset.NewProcessor(l, ctx, db)
		cp := compartment.NewProcessor(l, ctx, db)

		a, err := ap.GetByReferenceId(e.Id, asset.ReferenceTypeEquipable)
		if err != nil {
			return
		}
		c, err := cp.GetById(a.CompartmentId())
		if err != nil {
			return
		}
		_ = ap.RelayUpdateAndEmit(uuid.New(), c.CharacterId(), a.ReferenceId(), a.ReferenceType(), a.ReferenceData())
	}
}

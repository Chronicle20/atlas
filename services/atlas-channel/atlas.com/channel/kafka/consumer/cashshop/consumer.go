package cashshop

import (
	"atlas-channel/cashshop/wallet"
	consumer2 "atlas-channel/kafka/consumer"
	cashshop2 "atlas-channel/kafka/message/cashshop"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("cash_shop_status_event")(cashshop2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(cashshop2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventInventoryCapacityIncreased(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventError(sc, wp))))
			}
		}
	}
}

func handleStatusEventInventoryCapacityIncreased(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.InventoryCapacityIncreasedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.InventoryCapacityIncreasedBody]) {
		if e.Type != cashshop2.StatusEventTypeInventoryCapacityIncreased {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			err := session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopInventoryCapacityIncreaseSuccessBody(l)(e.Body.InventoryType, e.Body.Capacity))(s)
			if err != nil {
				return err
			}
			w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
				w = wallet.Model{}
			}
			err = session.Announce(l)(ctx)(wp)(writer.CashShopCashQueryResult)(writer.CashShopCashQueryResultBody(t)(w.Credit(), w.Points(), w.Prepaid()))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce cash shop wallet to character [%d].", s.CharacterId())
				return err
			}
			return nil
		})
		return
	}
}

func handleStatusEventError(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.ErrorEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.ErrorEventBody]) {
		if e.Type != cashshop2.StatusEventTypeError {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		// TODO this is not a generic error generator
		op := session.Announce(l)(ctx)(wp)(writer.CashShopOperation)(writer.CashShopInventoryCapacityIncreaseFailedBody(l)(e.Body.Error))
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, op)
		return
	}
}

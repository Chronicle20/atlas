package wallet

import (
	consumer2 "atlas-channel/kafka/consumer"
	walletmsg "atlas-channel/kafka/message/wallet"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// InitConsumers registers the EVENT_TOPIC_WALLET_STATUS consumer (mirrors the
// cash-shop status-event consumer): tenant/span header parsers + start at the
// latest offset (status events are live notifications, not replayed).
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("wallet_status_event")(walletmsg.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

// InitHandlers wires the wallet-updated handler onto the status topic. The
// handler refreshes the on-screen NX/points counter for whichever cash scene
// (MTS or the regular cash shop) the character currently occupies, using the
// balances carried on the event itself rather than re-reading over REST.
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(walletmsg.EnvEventTopicStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleWalletUpdated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// handleWalletUpdated pushes the refreshed wallet balances to the character's
// session once the debit/credit has actually landed (EVENT_TOPIC_WALLET_STATUS
// UPDATED), instead of the racy pre-debit refresh the BID_PLACED/OUTBID MTS
// events used to trigger. Which clientbound packet is written depends on which
// cash scene the player is currently in (session.CashScene()): MTS gets
// MtsOperation2, the regular cash shop gets CashQueryResult. A player in
// neither scene has nothing to refresh.
func handleWalletUpdated(sc server.Model, wp writer.Producer) message.Handler[walletmsg.StatusEvent[walletmsg.StatusEventUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e walletmsg.StatusEvent[walletmsg.StatusEventUpdatedBody]) {
		if e.Type != walletmsg.StatusEventTypeUpdated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByAccountId(sc.Channel())(e.AccountId, func(s session.Model) error {
			switch s.CashScene() {
			case session.CashSceneMts:
				return session.Announce(l)(ctx)(wp)(fieldcb.MtsOperation2Writer)(fieldcb.NewMtsOperation2(e.Body.Prepaid, e.Body.Points).Encode)(s)
			case session.CashSceneCashShop:
				return session.Announce(l)(ctx)(wp)(cashcb.CashQueryResultWriter)(cashcb.NewCashQueryResult(e.Body.Credit, e.Body.Points, e.Body.Prepaid).Encode)(s)
			default:
				return nil
			}
		})
	}
}

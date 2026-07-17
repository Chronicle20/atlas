package teleportrock

import (
	consumer2 "atlas-channel/kafka/consumer"
	teleportrock2 "atlas-channel/kafka/message/teleportrock"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	trcb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("teleport_rock_status_event")(teleportrock2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(teleportrock2.EnvEventTopicStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleListUpdated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleError(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// handleListUpdated projects LIST_UPDATED into the list-refresh
// MAP_TRANSFER_RESULT (mode REGISTER_LIST on add, DELETE_LIST on remove). The
// client only updates its UI from this packet (FR-7).
func handleListUpdated(sc server.Model, wp writer.Producer) message.Handler[teleportrock2.StatusEvent[teleportrock2.ListUpdatedStatusBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e teleportrock2.StatusEvent[teleportrock2.ListUpdatedStatusBody]) {
		if e.Type != teleportrock2.StatusEventTypeListUpdated {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}
		key := trpkt.MapTransferModeDeleteList
		if e.Body.Registered {
			key = trpkt.MapTransferModeRegisterList
		}
		body := trpkt.MapTransferResultListBody(key, e.Body.Vip, e.Body.Maps)
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(trcb.MapTransferResultWriter)(body))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce teleport-rock list update to character [%d].", e.CharacterId)
		}
	}
}

// handleError maps rejection reasons to the faithful client strings (design
// §4.2): LIST_FULL/DUPLICATE/MAP_NOT_ALLOWED -> MAP_NOT_AVAILABLE (the client
// prechecks full/duplicate itself; these fire only for bypassed clients);
// NOT_FOUND -> CANNOT_GO.
func handleError(sc server.Model, wp writer.Producer) message.Handler[teleportrock2.StatusEvent[teleportrock2.ErrorStatusBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e teleportrock2.StatusEvent[teleportrock2.ErrorStatusBody]) {
		if e.Type != teleportrock2.StatusEventTypeError {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}
		key := errorReasonToModeKey(e.Body.Reason)
		body := trpkt.MapTransferResultErrorBody(key, e.Body.Vip)
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(trcb.MapTransferResultWriter)(body))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce teleport-rock error to character [%d].", e.CharacterId)
		}
	}
}

func errorReasonToModeKey(reason string) string {
	switch reason {
	case teleportrock2.ErrorReasonNotFound:
		return trpkt.MapTransferModeCannotGo
	default: // LIST_FULL, DUPLICATE, MAP_NOT_ALLOWED
		return trpkt.MapTransferModeMapNotAvailable
	}
}

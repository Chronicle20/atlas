package saga

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/kafka/message/saga"
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
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	storagepkt "github.com/Chronicle20/atlas/libs/atlas-packet/storage"
	storagecb "github.com/Chronicle20/atlas/libs/atlas-packet/storage/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// InitConsumers initializes saga status event consumers
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("saga_status_event")(saga.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

// InitHandlers initializes saga status event handlers
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCompletedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleFailedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// mtsTakeHomePurchaseTab / mtsTakeHomeSelectedNo mirror the values the MTS
// status consumer's handleItemTakenHome passes to MoveItcPurchaseItemLtoSDone:
// the purchase ("taken home") items live in the first MTS tab (tab=1 -> index 0
// via SetTab(tab-1)); selectedNo 0 leaves the selection at the top of the list.
const (
	mtsTakeHomePurchaseTab uint32 = 1
	mtsTakeHomeSelectedNo  uint32 = 0
)

// handleCompletedEvent handles saga completion events
func handleCompletedEvent(sc server.Model, wp writer.Producer) message.Handler[saga.StatusEvent[saga.StatusEventCompletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
		if e.Type != saga.StatusEventTypeCompleted {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Saga transaction [%s] completed successfully (type [%s]).", e.TransactionId.String(), e.Body.SagaType)

		// Take-home (WithdrawFromMts) completion: the item has actually been granted
		// to the character's inventory (this fires from the orchestrator's single
		// guarded terminal-completion emit, AFTER both release + accept_to_character
		// succeeded). Write MoveItcPurchaseItemLtoSDone to the originating session so
		// the seller/buyer's take-home UI unhangs. A failed/compensated saga never
		// reaches COMPLETED, so this is only ever sent on real success.
		if e.Body.SagaType == saga.SagaTypeMtsOperation && resultKind(e.Body.Results) == saga.MtsTakeHomeResultKind {
			characterId := resultUint32(e.Body.Results, "characterId")
			if characterId == 0 {
				l.WithField("transaction_id", e.TransactionId.String()).Warn("MTS take-home completion missing characterId; cannot notify session.")
				return
			}
			announceMtsTakeHomeDone(l, ctx, sc, wp, characterId)
			return
		}

		// Storage mesos update is handled by storage consumer; the character sees
		// other results through their respective domain events.
	}
}

// announceMtsTakeHomeDone resolves the character's session on this channel and
// writes the MtsOperation MoveItcPurchaseItemLtoSDone result. A missing session
// (character not on this channel) is a graceful no-op.
func announceMtsTakeHomeDone(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32) {
	s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(characterId)
	if err != nil {
		l.WithField("character_id", characterId).Debug("Character not connected, skipping MTS take-home notification.")
		return
	}
	if s.ChannelId() != sc.ChannelId() {
		return
	}
	if err := session.Announce(l)(ctx)(wp)(fieldcb.MtsOperationWriter)(fieldpkt.MtsOperationMoveItcPurchaseItemLtoSDoneBody(mtsTakeHomePurchaseTab, mtsTakeHomeSelectedNo))(s); err != nil {
		l.WithError(err).WithField("character_id", characterId).Error("Failed to send MTS take-home done packet to client.")
	}
}

// resultKind reads the "kind" marker off a saga COMPLETED Results map.
func resultKind(results map[string]any) string {
	if results == nil {
		return ""
	}
	if v, ok := results["kind"].(string); ok {
		return v
	}
	return ""
}

// resultUint32 reads a uint32 off a saga COMPLETED Results map, tolerating the
// float64 the value becomes after a JSON round-trip.
func resultUint32(results map[string]any, key string) uint32 {
	if results == nil {
		return 0
	}
	switch v := results[key].(type) {
	case float64:
		return uint32(v)
	case uint32:
		return v
	case int:
		return uint32(v)
	default:
		return 0
	}
}

// handleFailedEvent handles saga failure events
func handleFailedEvent(sc server.Model, wp writer.Producer) message.Handler[saga.StatusEvent[saga.StatusEventFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventFailedBody]) {
		if e.Type != saga.StatusEventTypeFailed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"saga_type":      e.Body.SagaType,
			"error_code":     e.Body.ErrorCode,
			"character_id":   e.Body.CharacterId,
			"failed_step":    e.Body.FailedStep,
		}).Debugf("Saga transaction failed. Reason: [%s]", e.Body.Reason)

		// Look up the session for the character
		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.Body.CharacterId)
		if err != nil {
			l.WithField("character_id", e.Body.CharacterId).Debug("Character not connected, skipping error notification.")
			return
		}

		if s.ChannelId() != sc.ChannelId() {
			return
		}

		// Handle storage operation failures by sending appropriate error packets
		if e.Body.SagaType == saga.SagaTypeStorageOperation {
			// Get the appropriate error body producer based on the error code
			errorBody := getStorageErrorBodyProducer(e.Body.ErrorCode)
			if errorBody == nil {
				l.WithField("error_code", e.Body.ErrorCode).Warn("No error body producer for error code, skipping notification.")
				return
			}

			// Send the error packet to the client
			err = session.Announce(l)(ctx)(wp)(storagecb.StorageOperationWriter)(errorBody)(s)
			if err != nil {
				l.WithError(err).WithField("character_id", e.Body.CharacterId).Error("Failed to send storage error packet to client.")
				return
			}

			l.WithFields(logrus.Fields{
				"character_id": e.Body.CharacterId,
				"error_code":   e.Body.ErrorCode,
			}).Debug("Sent storage operation error packet to client.")
		}
	}
}

// getStorageErrorBodyProducer returns the appropriate BodyProducer for the given error code
func getStorageErrorBodyProducer(errorCode string) packet.Encode {
	switch errorCode {
	case saga.ErrorCodeNotEnoughMesos:
		return storagepkt.StorageOperationErrorNotEnoughMesoBody()
	case saga.ErrorCodeInventoryFull, saga.ErrorCodeStorageFull:
		return storagepkt.StorageOperationErrorInventoryFullBody()
	default:
		return nil
	}
}

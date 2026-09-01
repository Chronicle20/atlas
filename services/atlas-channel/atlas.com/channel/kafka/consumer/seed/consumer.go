// Package seed consumes atlas-character-factory's seed-status events
// (EVENT_TOPIC_SEED_STATUS) for a Maple Life (Cash/0543) character creation.
// The submit handler (socket/handler/maple_life_create.go) creates the
// character through the factory FIRST and records a PhaseSubmitted entry in
// the maplelife registry; this package closes the loop from the other side --
// on CREATED it destroys the cash item that paid for the creation (FR-5.1),
// and on FAILED it leaves the item intact.
//
// This is the only place in the classification-543 flow permitted to create
// a destroy saga (task-246 design §5.4): every gate the submit handler runs
// leaves the item untouched, so consumption happens exactly once, only after
// the character genuinely exists.
package seed

import (
	consumer2 "atlas-channel/kafka/consumer"
	seedmsg "atlas-channel/kafka/message/seed"
	"atlas-channel/listener"
	"atlas-channel/maplelife"
	"atlas-channel/saga"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	mlcb "github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// destroyCashItemFunc is the test seam over saga.Processor.Create (package-var
// injection precedent: seedCharacterFunc in socket/handler/maple_life_create.go,
// cashItemInSlotFunc in character_cash_item_use.go). The caller builds the
// whole saga.Saga -- exactly as character_cash_item_use.go:626-637 builds its
// consume_sacrifice step -- so a test substituting this seam can assert on
// the saga it was actually given, not just the arguments used to build it.
var destroyCashItemFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

// buildDestroyCashItemSaga builds the one-step MAPLELIFE_USE saga that
// destroys the cash item paying for a Maple Life character now that
// atlas-character-factory has reported CREATED. Not RequestItemConsume --
// that routes through atlas-consumables' *use* semantics (effects,
// cooldowns); the item is destroyed because a purchase was fulfilled, exactly
// as character_cash_item_use.go's consume_sacrifice step destroys an
// incubator sacrifice.
func buildDestroyCashItemSaga(characterId uint32, invType byte, slot int16, templateId uint32) saga.Saga {
	now := time.Now()
	return saga.Saga{
		TransactionId: uuid.New(),
		SagaType:      saga.MapleLifeUse,
		InitiatedBy:   "MAPLE_LIFE_USE",
		Steps: []saga.Step{
			{
				StepId: "destroy_maple_life_item",
				Status: saga.Pending,
				Action: saga.DestroyAssetFromSlot,
				Payload: saga.DestroyAssetFromSlotPayload{
					CharacterId:   characterId,
					InventoryType: invType,
					Slot:          slot,
					Quantity:      1,
					TemplateId:    templateId,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// InitConsumers registers the EVENT_TOPIC_SEED_STATUS consumer, mirroring the
// cash-shop/wallet status-event consumers (latest offset -- a missed status
// event is bridged by maplelife.Registry.Sweep's SubmittedTTL, not replay).
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("seed_status_event")(seedmsg.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

// InitHandlers wires the CREATED and FAILED seed-status handlers onto the
// status topic.
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var err error
				var handles []listener.HandlerHandle
				t, err = topic.EnvProvider(l)(seedmsg.EnvEventTopicStatus)()
				if err != nil {
					return nil, err
				}
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedStatusEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleFailedStatusEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// resolveEntry finds the pending PhaseSubmitted entry a seed status event
// correlates to. TransactionId is resolved first when the event carries one:
// a TransactionId present on the event but matching nothing is a mismatch --
// logged and dropped by the caller, never a fallback to account id, or the
// correlation guarantee buys nothing. Only an EMPTY TransactionId (an older
// factory build mid-rollout, design §4.3) falls back to Take by account id.
func resolveEntry(t tenant.Model, accountId uint32, transactionId string) (uint32, maplelife.Entry, bool) {
	if transactionId != "" {
		return maplelife.GetRegistry().TakeByTransactionId(t, transactionId)
	}
	e, ok := maplelife.GetRegistry().Take(t, accountId)
	return accountId, e, ok
}

// handleCreatedStatusEvent destroys the Maple Life cash item that paid for
// the now-created character (FR-5.1), then announces MAPLELIFE_ERROR's
// SUCCESS arm to the session that submitted it, if still connected.
//
// entry.CharacterId (from the registry) is the SUBMITTING character -- the
// one whose cash inventory holds the item, and the only id the destroy step
// may carry. e.Body.CharacterId is the NEWLY CREATED character; it never
// reaches the destroy step and is used for logging/correlation only.
func handleCreatedStatusEvent(sc server.Model, wp writer.Producer) message.Handler[seedmsg.StatusEvent[seedmsg.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e seedmsg.StatusEvent[seedmsg.CreatedStatusEventBody]) {
		if e.Type != seedmsg.StatusEventTypeCreated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		accountId, entry, ok := resolveEntry(t, e.AccountId, e.TransactionId)
		if !ok {
			l.WithFields(logrus.Fields{
				"account_id":      e.AccountId,
				"transaction_id":  e.TransactionId,
				"created_char_id": e.Body.CharacterId,
			}).Warn("Seed CREATED did not correlate to any pending Maple Life submission; dropping.")
			return
		}

		destroySaga := buildDestroyCashItemSaga(entry.CharacterId, byte(inventory.TypeValueCash), int16(entry.Slot), uint32(entry.ItemId))
		if err := destroyCashItemFunc(l, ctx, destroySaga); err != nil {
			// Rolling a created character back to reclaim a cash item is
			// destructive and disproportionate (design §5.4): log at ERROR
			// and stop. No compensating character deletion is attempted.
			l.WithError(err).WithFields(logrus.Fields{
				"account_id":            accountId,
				"submitting_char_id":    entry.CharacterId,
				"created_char_id":       e.Body.CharacterId,
				"item_id":               entry.ItemId,
				"submit_transaction_id": entry.TransactionId,
			}).Error("Unable to destroy the Maple Life cash item after character creation; the item was not reclaimed.")
		}

		found := false
		_ = session.NewProcessor(l, ctx).IfPresentByAccountId(sc.Channel())(accountId, func(s session.Model) error {
			found = true
			return session.Announce(l)(ctx)(wp)(mlcb.MapleLifeErrorWriter)(mlcb.MapleLifeErrorBody(mlcb.MapleLifeErrorSuccess))(s)
		})
		if !found {
			// The entitlement was already spent and the character exists;
			// leaving the item would let one item produce two characters
			// (design §5.4). The destroy above still happened -- only the
			// client write is skipped.
			l.WithFields(logrus.Fields{
				"account_id":      accountId,
				"created_char_id": e.Body.CharacterId,
			}).Info("Seed CREATED received for disconnected session; item reclaimed, no client write.")
		}
	}
}

// handleFailedStatusEvent announces MAPLELIFE_ERROR's generic UNKNOWN_ERROR
// arm and touches no item -- the cash item survives every saga failure by
// construction (design §5.4).
func handleFailedStatusEvent(sc server.Model, wp writer.Producer) message.Handler[seedmsg.StatusEvent[seedmsg.FailedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e seedmsg.StatusEvent[seedmsg.FailedStatusEventBody]) {
		if e.Type != seedmsg.StatusEventTypeFailed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		accountId, entry, ok := resolveEntry(t, e.AccountId, e.TransactionId)
		if !ok {
			l.WithFields(logrus.Fields{
				"account_id":     e.AccountId,
				"transaction_id": e.TransactionId,
				"reason":         e.Body.Reason,
			}).Warn("Seed FAILED did not correlate to any pending Maple Life submission; dropping.")
			return
		}

		l.WithFields(logrus.Fields{
			"account_id":     accountId,
			"item_id":        entry.ItemId,
			"transaction_id": entry.TransactionId,
			"reason":         e.Body.Reason,
		}).Info("Maple Life character creation failed; leaving item intact.")

		found := false
		_ = session.NewProcessor(l, ctx).IfPresentByAccountId(sc.Channel())(accountId, func(s session.Model) error {
			found = true
			return session.Announce(l)(ctx)(wp)(mlcb.MapleLifeErrorWriter)(mlcb.MapleLifeErrorBody(mlcb.MapleLifeErrorUnknownError))(s)
		})
		if !found {
			l.WithFields(logrus.Fields{
				"account_id": accountId,
			}).Info("Seed FAILED received for disconnected session; dropping.")
		}
	}
}

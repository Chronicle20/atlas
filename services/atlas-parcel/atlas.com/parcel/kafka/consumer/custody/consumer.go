package custody

import (
	"atlas-parcel/kafka/consumer"
	buffer "atlas-parcel/kafka/message"
	"atlas-parcel/kafka/message/custody"
	parcelmsg "atlas-parcel/kafka/message/parcel"
	custodyproducer "atlas-parcel/kafka/producer/custody"
	parcelproducer "atlas-parcel/kafka/producer/parcel"
	"atlas-parcel/parcel"
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	kconsumer "github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// InitConsumers registers the parcel custody command consumer (the saga
// custody channel), mirroring services/atlas-mts/atlas.com/mts/kafka/consumer/custody/consumer.go.
func InitConsumers(l logrus.FieldLogger) func(func(config kconsumer.Config, decorators ...model.Decorator[kconsumer.Config])) func(consumerGroupId string) {
	return func(rf func(config kconsumer.Config, decorators ...model.Decorator[kconsumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer.NewConfig(l)("parcel_custody_command")(custody.EnvCommandTopic)(consumerGroupId), kconsumer.SetHeaderParsers(kconsumer.SpanHeaderParser, kconsumer.TenantHeaderParser, kconsumer.EnvHeaderParser))
		}
	}
}

// InitHandlers wires the accept/release/restore/remove custody command
// handlers onto the custody command topic. The producer.Provider is
// constructed per delivery from the message context so emitted acks carry
// the right tenant/span headers.
func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			var err error
			t, err = topic.EnvProvider(l)(custody.EnvCommandTopic)()
			if err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptToParcel(producer.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleReleaseFromParcel(producer.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRestoreParcel(producer.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoveParcel(producer.ProviderImpl(l))(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// providerFn is the shape of the per-context producer factory returned by
// producer.ProviderImpl(l): func(ctx) func(token) MessageProducer.
type providerFn = func(ctx context.Context) producer.Provider

// processor type-asserts down to *parcel.ProcessorImpl, whose custody methods
// (AcceptCustody/ReleaseCustody/RestoreCustody/RemoveCustody) are additive to
// the Processor interface — parcel.Processor did not need to change to
// support Task 15 (see processor.go's package doc comment).
func processor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *parcel.ProcessorImpl {
	return parcel.NewProcessor(l, ctx, db).(*parcel.ProcessorImpl)
}

// handleAcceptToParcel CREATES the pending parcel row from the carried
// snapshot, using the caller-supplied ParcelId so the create is deterministic
// and idempotent on replay. A replayed delivery (same ParcelId) finds the row
// already present and re-acks ACCEPTED without creating a duplicate.
func handleAcceptToParcel(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.AcceptToParcelCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.AcceptToParcelCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.AcceptToParcelCommandBody]) {
			if c.Type != custody.CommandAcceptToParcel {
				return
			}
			b := c.Body
			p := pf(ctx)

			err := buffer.Emit(p)(func(mb *buffer.Buffer) error {
				m, aerr := processor(l, ctx, db).AcceptCustody(parcel.AcceptParams{
					ParcelId:           b.ParcelId,
					CharacterId:        b.CharacterId,
					WorldId:            b.WorldId,
					SenderAccountId:    b.SenderAccountId,
					SenderName:         b.SenderName,
					RecipientId:        b.RecipientId,
					RecipientAccountId: b.RecipientAccountId,
					RecipientName:      b.RecipientName,
					MesoAmount:         b.MesoAmount,
					FeePaid:            b.FeePaid,
					Quick:              b.Quick,
					Message:            b.Message,
					ReceivableAt:       b.ReceivableAt,
					ExpiresAt:          b.ExpiresAt,
					HasItem:            b.HasItem,
					ItemType:           b.ItemType,
					TemplateId:         b.TemplateId,
					Quantity:           b.Quantity,
					Strength:           b.Strength,
					Dexterity:          b.Dexterity,
					Intelligence:       b.Intelligence,
					Luck:               b.Luck,
					HP:                 b.HP,
					MP:                 b.MP,
					WeaponAttack:       b.WeaponAttack,
					MagicAttack:        b.MagicAttack,
					WeaponDefense:      b.WeaponDefense,
					MagicDefense:       b.MagicDefense,
					Accuracy:           b.Accuracy,
					Avoidability:       b.Avoidability,
					Hands:              b.Hands,
					Speed:              b.Speed,
					Jump:               b.Jump,
					Slots:              b.Slots,
					Level:              b.Level,
					ItemLevel:          b.ItemLevel,
					ItemExp:            b.ItemExp,
					RingId:             b.RingId,
					ViciousCount:       b.ViciousCount,
					Flags:              b.Flags,
					Owner:              b.Owner,
				})
				if aerr != nil {
					return aerr
				}
				if perr := mb.Put(custody.EnvStatusTopic, custodyproducer.AcceptedStatusEventProvider(c.TransactionId, m.Id())); perr != nil {
					return perr
				}
				// accept_to_parcel is the LAST step of parcel_send, so this
				// ack is also the sender's "it went out" signal: tell the
				// sender's channel so it can announce
				// PARCEL[SUCCESSFULLY_SENT] and the client re-enables its
				// send tab. b.CharacterId is the sender (the saga's
				// AcceptToParcelPayload.CharacterId comes from the sending
				// session), not the recipient. A replayed delivery re-emits
				// the notice — harmless, and cheaper than tracking notice
				// state on a row whose create is already idempotent.
				return mb.Put(parcelmsg.EnvStatusEventTopic, parcelproducer.ParcelSentStatusEventProvider(b.CharacterId))
			})
			if err != nil {
				l.WithError(err).Errorf("Failed to accept parcel [%s] for transaction [%s].", b.ParcelId.String(), c.TransactionId.String())
				_ = p(custody.EnvStatusTopic)(custodyproducer.ErrorStatusEventProvider(c.TransactionId, err.Error()))
				return
			}
		}
	}
}

// handleReleaseFromParcel transitions the parcel row to received AND
// releases custody in one atlas-parcel transaction (design §4.3). A
// replayed delivery is not an error (parcel.ErrAlreadyReleased) — it re-acks
// nothing, since the first delivery already emitted the RELEASED event (and,
// per below, the PARCEL_RECEIVED player-facing event too).
//
// The release IS the completion — the direct analogue of
// handleAcceptToParcel's PARCEL_SENT emission ("the row create IS the
// completion"). If the downstream accept_to_character later fails and
// handleRestoreParcel compensates, the client has already removed the row
// from PARCEL_RECEIVED and will only see it again on reopening the dialog.
// Accepted trade-off (bug-duey-receive-no-completion-confirmation.md's "Not
// yet answered"); the fix if this proves regular in live testing is a re-add
// packet from handleRestoreParcel, not moving where this is emitted.
func handleReleaseFromParcel(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.ReleaseFromParcelCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.ReleaseFromParcelCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.ReleaseFromParcelCommandBody]) {
			if c.Type != custody.CommandReleaseFromParcel {
				return
			}
			b := c.Body
			p := pf(ctx)

			err := buffer.Emit(p)(func(mb *buffer.Buffer) error {
				m, rerr := processor(l, ctx, db).ReleaseCustody(b.ParcelId, b.RecipientId)
				if rerr != nil {
					return rerr
				}
				if perr := mb.Put(custody.EnvStatusTopic, custodyproducer.ReleasedStatusEventProvider(c.TransactionId, m.Id())); perr != nil {
					return perr
				}
				return mb.Put(parcelmsg.EnvStatusEventTopic, parcelproducer.ParcelReceivedStatusEventProvider(b.RecipientId, m.Id()))
			})
			if errors.Is(err, parcel.ErrAlreadyReleased) {
				l.Infof("ReleaseFromParcel: parcel [%s] already released; replay is a no-op, transaction [%s].", b.ParcelId.String(), c.TransactionId.String())
				return
			}
			if err != nil {
				l.WithError(err).Errorf("Failed to release parcel [%s] for transaction [%s].", b.ParcelId.String(), c.TransactionId.String())
				_ = p(custody.EnvStatusTopic)(custodyproducer.ErrorStatusEventProvider(c.TransactionId, err.Error()))
				return
			}
		}
	}
}

// handleRestoreParcel un-resolves a parcel released by handleReleaseFromParcel
// whose downstream accept_to_character then failed — the compensating
// inverse, dispatched fire-and-forget by the orchestrator's compensator. No
// consumer awaits a success ack; only a failure emits an ERROR event (mirrors
// the MTS late-inverse handlers' emit-on-error-only rationale).
func handleRestoreParcel(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.RestoreParcelCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.RestoreParcelCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.RestoreParcelCommandBody]) {
			if c.Type != custody.CommandRestoreParcel {
				return
			}
			b := c.Body

			if err := processor(l, ctx, db).RestoreCustody(b.ParcelId); err != nil {
				l.WithError(err).Errorf("Failed to restore parcel [%s] for transaction [%s].", b.ParcelId.String(), c.TransactionId.String())
				_ = pf(ctx)(custody.EnvStatusTopic)(custodyproducer.ErrorStatusEventProvider(c.TransactionId, err.Error()))
				return
			}
			l.Infof("RestoreParcel: restored parcel [%s] (or already pending), transaction [%s].", b.ParcelId.String(), c.TransactionId.String())
		}
	}
}

// handleRemoveParcel hard-deletes a still-pending parcel row created by a
// late ACCEPT_TO_PARCEL after its saga already compensated — the compensating
// inverse, dispatched fire-and-forget by the orchestrator's compensator. No
// consumer awaits a success ack; only a failure emits an ERROR event.
func handleRemoveParcel(pf providerFn) func(db *gorm.DB) message.Handler[custody.Command[custody.RemoveParcelCommandBody]] {
	return func(db *gorm.DB) message.Handler[custody.Command[custody.RemoveParcelCommandBody]] {
		return func(l logrus.FieldLogger, ctx context.Context, c custody.Command[custody.RemoveParcelCommandBody]) {
			if c.Type != custody.CommandRemoveParcel {
				return
			}
			b := c.Body

			if err := processor(l, ctx, db).RemoveCustody(b.ParcelId); err != nil {
				l.WithError(err).Errorf("Failed to remove parcel [%s] for transaction [%s].", b.ParcelId.String(), c.TransactionId.String())
				_ = pf(ctx)(custody.EnvStatusTopic)(custodyproducer.ErrorStatusEventProvider(c.TransactionId, err.Error()))
				return
			}
			l.Infof("RemoveParcel: removed parcel [%s] (or not pending), transaction [%s].", b.ParcelId.String(), c.TransactionId.String())
		}
	}
}

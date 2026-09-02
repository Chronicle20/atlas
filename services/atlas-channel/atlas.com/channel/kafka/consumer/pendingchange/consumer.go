// Package pendingchange delivers a resolved PENDING_CHANGE (design §3.9) to
// an online player as the version-appropriate CANCEL_* packet plus a
// pink-text world message — the belt-and-braces answer to OQ-9: if the
// client ignores the CANCEL_* packet outside the cash-shop UI, the player
// still learns why their coupon came back.
package pendingchange

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	chatcb "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("pending_change_event")(EnvEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var err error
				var handles []listener.HandlerHandle
				t, err = topic.EnvProvider(l)(EnvEventTopic)()
				if err != nil {
					return nil, err
				}
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleResolved(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// handleResolved is FR-2.9 / FR-2.7's delivery half. APPLIED is not a
// cancellation and produces neither packet nor pink text. Every other
// terminal status (CANCELLED, REJECTED, EXPIRED) writes exactly one CANCEL_*
// packet, chosen by (ChangeType, Status, Reason), followed by a pink-text
// notice naming the requested value.
//
// When no live session is found for the character (offline, or this pod does
// not host their channel), IfPresentByCharacterId is a pure no-op: nothing is
// written. atlas-character's notified_at is never stamped by this consumer —
// only its own LOGIN catch-up (RenotifyForCharacter, Task 9) advances it — so
// an offline delivery here costs nothing and loses nothing; the catch-up
// re-emits at the player's next login.
func handleResolved(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[ResolvedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[ResolvedEventBody]) {
		if e.Type != EventTypeResolved {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, e.WorldId) {
			return
		}

		b := e.Body

		switch b.Status {
		case StatusCancelled, StatusRejected, StatusExpired:
			// terminal, non-applied — proceed below
		default:
			// StatusApplied is a success, not a cancellation, and must
			// produce neither packet nor pink text. Any other value
			// (StatusPending, or an unrecognized future status) is not a
			// resolution this consumer knows how to notify and is silently
			// ignored rather than guessed at.
			return
		}

		var writerName string
		var body packet.Encode
		switch b.ChangeType {
		case ChangeTypeNameChange:
			if b.Status == StatusRejected && b.Reason == ReasonNameTaken {
				// FR-2.7: invalidated because someone else took the name.
				// CANCEL_NAME_CHANGE_BY_OTHER's wire body is EMPTY (the
				// client renders one fixed StringPool notice) — the pink
				// text below is the ONLY way the player learns which name
				// was lost, not belt-and-braces.
				writerName = charcb.CancelNameChangeByOtherWriter
				body = charcb.NewCancelNameChangeByOther().Encode
			} else {
				writerName = cashcb.CashShopCancelNameChangeResultWriter
				body = cashcb.CancelNameChangeResultCancelledBody()
			}
		case ChangeTypeWorldTransfer:
			writerName = cashcb.CashShopCancelTransferWorldResultWriter
			body = cashcb.CancelTransferWorldResultCancelledBody()
		default:
			l.Warnf("Pending change [%s] for character [%d] resolved with unrecognized changeType [%s]; no notification sent.", b.PendingChangeId, e.CharacterId, b.ChangeType)
			return
		}

		msg := resolutionPinkText(b)

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			// The CANCEL_* packet is attempted first, but its absence from a
			// tenant's template (gms_v48 has no name-change receiver at all;
			// jms_v185 has no name-change feature whatsoever — both are
			// correct, derived exclusions, not gaps) must not suppress the
			// pink text: on every version the pink text is what the belt-
			// and-braces design (§3.9) exists to guarantee.
			if aErr := session.Announce(l)(ctx)(wp)(writerName)(body)(s); aErr != nil {
				l.WithError(aErr).Debugf("Unable to write [%s] for character [%d]; tenant template likely has no binding for this writer (expected on some versions). Pink text will still be sent.", writerName, e.CharacterId)
			}
			return session.Announce(l)(ctx)(wp)(chatcb.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", "", msg))(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to deliver pending-change resolution pink text for character [%d].", e.CharacterId)
		}
	}
}

// resolutionPinkText names the requested value so the notice is
// self-explanatory even when the accompanying packet carries no message of
// its own (CANCEL_NAME_CHANGE_BY_OTHER's empty body).
func resolutionPinkText(b ResolvedEventBody) string {
	switch b.ChangeType {
	case ChangeTypeNameChange:
		return fmt.Sprintf("Your name change to \"%s\" was not applied and has been cancelled.", b.RequestedName)
	case ChangeTypeWorldTransfer:
		return fmt.Sprintf("Your world transfer to world [%d] was not applied and has been cancelled.", b.DestinationWorldId)
	default:
		return "A pending change to your character was cancelled."
	}
}

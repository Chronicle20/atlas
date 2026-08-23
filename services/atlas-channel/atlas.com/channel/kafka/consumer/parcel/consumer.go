package parcel

import (
	consumer2 "atlas-channel/kafka/consumer"
	parcelmsg "atlas-channel/kafka/message/parcel"
	"atlas-channel/listener"
	dueyparcel "atlas-channel/parcel"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	atlasmodel "github.com/Chronicle20/atlas/libs/atlas-model/model"
	packetparcel "github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
	parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...atlasmodel.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...atlasmodel.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("parcel_show_command")(parcelmsg.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
			rf(consumer2.NewConfig(l)("parcel_status_event")(parcelmsg.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				t, _ := topic.EnvProvider(l)(parcelmsg.EnvCommandTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleShowParcelCommand(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles := []listener.HandlerHandle{{Topic: t, Id: id}}

				t, _ = topic.EnvProvider(l)(parcelmsg.EnvStatusEventTopic)()
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleParcelArrivedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleParcelSentEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleParcelReceivedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

				return handles, nil
			}
		}
	}
}

// showParcelDeps collects the atlas-parcel collaborators showParcel needs,
// mirroring socket/handler's dueyReceiveDeps so the mailbox/arrived split
// and the notify stamp can be exercised without a REST client.
type showParcelDeps struct {
	getMailbox   func(recipientId uint32, worldId world.Id) ([]dueyparcel.Model, error)
	markNotified func(id uuid.UUID) error
}

// handleShowParcelCommand guards on tenant and — since the command is
// dispatched to one specific channel — on this instance actually being that
// channel (sc.Is), then announces through IfPresentByCharacterId, so a
// recipient with no session on this channel is a silent no-op (the same
// shape as the frederick notification handler,
// kafka/consumer/merchant/consumer.go:454's tenant guard +
// IfPresentByCharacterId no-op).
func handleShowParcelCommand(sc server.Model, wp writer.Producer) message.Handler[parcelmsg.ShowParcelCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, e parcelmsg.ShowParcelCommand) {
		if e.Type != parcelmsg.CommandTypeShowParcel {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		if !sc.Is(t, e.WorldId, e.ChannelId) {
			return
		}

		deps := showParcelDeps{
			getMailbox: func(recipientId uint32, worldId world.Id) ([]dueyparcel.Model, error) {
				return dueyparcel.NewProcessor(l, ctx).GetForRecipient(recipientId, worldId)
			},
			markNotified: func(id uuid.UUID) error {
				return dueyparcel.NewProcessor(l, ctx).MarkNotified(id)
			},
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, showParcel(l, ctx, wp, e, deps))
		if err != nil {
			l.WithError(err).Errorf("Unable to show parcel dialog to character [%d].", e.CharacterId)
		}
	}
}

// showParcel builds and announces the OPEN/OPEN_QUICK body. Quick short-
// circuits before any mailbox fetch (design §5.2/§9.5 — OPEN_QUICK carries
// no list at all).
//
// The non-quick path passes receiveOnly=FALSE. That bool is the client's
// CParcelDlg m_nMode, not a "quick delivery is available" flag: false
// builds CParcelDlg(0) with all three tabs (Receive + Send + QuickSend),
// true builds CParcelDlg(2), a Receive-only window with no way to send at
// all (parcel/clientbound/parcel.go's Open doc records the IDA addresses
// on v83 and v95). The NPC entry point is the full dialog, so it is false
// unconditionally — there is no tenant condition to consult.
//
// The mailbox excludes parcels not yet receivable (FR-12 — the client shows
// those with a countdown from a different surface, not Duey's OPEN list;
// design §5.3 makes no mention of a not-yet-receivable entry in the OPEN
// body). Of the receivable parcels, every one with a nil LastNotified is
// both included in the "new arrivals" second list AND stamped notified —
// the cheapest correct implementation of FR-24 (design §5.3), needing no
// extra packet. A stamp failure is logged and otherwise ignored: it only
// risks a parcel appearing in "new arrivals" again next open, never data
// loss.
func showParcel(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, e parcelmsg.ShowParcelCommand, deps showParcelDeps) func(s session.Model) error {
	return func(s session.Model) error {
		if e.Quick {
			return session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(parcelcb.ParcelOpenQuickBody())(s)
		}

		ms, err := deps.getMailbox(e.CharacterId, e.WorldId)
		if err != nil {
			return err
		}

		now := time.Now()
		var mailbox []packetparcel.Parcel
		var arrived []packetparcel.Parcel
		var toNotify []uuid.UUID
		for _, m := range ms {
			if m.ReceivableAt().After(now) {
				continue
			}
			p := m.ToPacket()
			mailbox = append(mailbox, p)
			if m.LastNotified() == nil {
				arrived = append(arrived, p)
				toNotify = append(toNotify, m.Id())
			}
		}

		for _, id := range toNotify {
			if err := deps.markNotified(id); err != nil {
				l.WithError(err).Warnf("Unable to stamp parcel [%s] notified.", id)
			}
		}

		return session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(parcelcb.ParcelOpenBody(false, mailbox, arrived))(s)
	}
}

// handleParcelArrivedEvent announces PARCEL[ALARM_NAMED] to a parcel's
// recipient when atlas-parcel publishes a PARCEL_ARRIVED status event
// (task-241 Task 24's producer). Guards on the event type and on tenant
// (t.Is(sc.Tenant())), mirroring handleFrederickNotificationEvent
// (kafka/consumer/merchant/consumer.go:454), then announces through
// IfPresentByCharacterId — a recipient with no session on this channel is a
// silent no-op.
//
// This always resolves to ALARM_NAMED (0x19), never PARCEL_ARRIVED (0x18).
// Design §7.1 makes this an explicit trade: 0x18 would both append the
// dialog row and raise SP_3902, but selecting it requires knowing the
// session has an open parcel dialog, which the channel does not track. A
// player with the dialog open sees a toast instead of a live row —
// accepted, low severity.
func handleParcelArrivedEvent(sc server.Model, wp writer.Producer) message.Handler[parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]) {
		if e.Type != parcelmsg.StatusEventParcelArrived {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(parcelcb.ParcelAlarmNamedBody(e.Body.SenderName, e.Body.HasItem)))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce parcel arrival to character [%d].", e.CharacterId)
		}
	}
}

// handleParcelSentEvent announces PARCEL[SUCCESSFULLY_SENT] (0x12) to a
// parcel's SENDER when atlas-parcel publishes PARCEL_SENT — the last step of
// the parcel_send saga having landed. Without it the client never learns the
// send finished: 0x12 is the arm that raises SP_3901 and, through
// CParcelDlg::OnPacket's default arm, calls SetCtrlEnabled(1) plus
// ResetSendInfo/CloseParcelDlg (v83 @0x6f579d), so the send tab stays
// greyed out until the dialog is reopened.
//
// Same guards and posture as handleParcelArrivedEvent: event type, tenant,
// then IfPresentByCharacterId — a sender who left the channel is a silent
// no-op.
func handleParcelSentEvent(sc server.Model, wp writer.Producer) message.Handler[parcelmsg.StatusEvent[parcelmsg.StatusEventParcelSentBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e parcelmsg.StatusEvent[parcelmsg.StatusEventParcelSentBody]) {
		if e.Type != parcelmsg.StatusEventParcelSent {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(parcelcb.ParcelSuccessfullySentBody()))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce parcel send completion to character [%d].", e.CharacterId)
		}
	}
}

// handleParcelReceivedEvent announces PARCEL[PARCEL_REMOVED] (kind Claimed)
// to a parcel's RECIPIENT when atlas-parcel publishes PARCEL_RECEIVED —
// handleReleaseFromParcel having completed. Without it the client never
// learns the receive finished: CParcelDlg disables its controls the moment
// it sends CTabReceive::ReceiveParcel and only re-enables them when a PARCEL
// result packet arrives (CParcelDlg::OnPacket, v83 @0x6f56ea). Case 23
// (PARCEL_REMOVED) calls RemoveParcel then SetCtrlEnabled(1) itself, so this
// one packet both removes the row and unlocks the dialog — no separate
// unlock packet is needed or sent.
//
// Same guards and posture as handleParcelSentEvent: event type, tenant, then
// IfPresentByCharacterId — a recipient who left the channel is a silent
// no-op. The parcel id is projected through dueyparcel.WireId, the same
// 4-byte big-endian truncation the OPEN list and RECEIVE resolution already
// agree on.
func handleParcelReceivedEvent(sc server.Model, wp writer.Producer) message.Handler[parcelmsg.StatusEvent[parcelmsg.StatusEventParcelReceivedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e parcelmsg.StatusEvent[parcelmsg.StatusEventParcelReceivedBody]) {
		if e.Type != parcelmsg.StatusEventParcelReceived {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(parcelcb.ParcelRemovedBody(dueyparcel.WireId(e.Body.ParcelId), parcelcb.ParcelRemovedKindClaimed)))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce parcel receive completion to character [%d].", e.CharacterId)
		}
	}
}

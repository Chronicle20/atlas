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
				return []listener.HandlerHandle{{Topic: t, Id: id}}, nil
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

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, showParcel(l, ctx, wp, t, e, deps))
		if err != nil {
			l.WithError(err).Errorf("Unable to show parcel dialog to character [%d].", e.CharacterId)
		}
	}
}

// showParcel builds and announces the OPEN/OPEN_QUICK body. Quick short-
// circuits before any mailbox fetch (design §5.2/§9.5 — OPEN_QUICK carries
// no list at all).
//
// For the non-quick path, quickEnabled reflects whether this tenant's
// client can even reach the Quick Delivery Ticket path — the same
// classification-533 version gate Task 22's handler uses to decide whether
// to open that dialog in the first place (quickDeliveryEnabled below);
// passed through rather than hard-coded true.
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
func showParcel(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, t tenant.Model, e parcelmsg.ShowParcelCommand, deps showParcelDeps) func(s session.Model) error {
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

		return session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(parcelcb.ParcelOpenBody(quickDeliveryEnabled(t), mailbox, arrived))(s)
	}
}

// quickDeliveryEnabled reports whether this tenant's client can reach the
// Quick Delivery Ticket path (classification 533). This is the same
// condition as socket/handler/character_cash_item_use_duey.go's
// dueyCouponEnabled, in a different package — the two packages can't share
// code across the boundary, so keep them in sync by hand.
//
// GMS half: t.IsRegion("GMS") && t.MajorAtLeast(72), mirroring
// remoteMerchantEnabled's identical gate for classification 545
// (socket/handler/character_cash_item_use_remote_merchant.go).
//
// JMS half: JMS v185 is enabled. Two facts settle it (task-241 task-22
// controller addendum, IDA-verified, MapleStory_dump_SCY.exe session
// 05eb9c27):
//  1. get_cashslot_item_type @0x49a1ee contains `case 533: return 32;` — JMS
//     routes classification 533 to cash-slot type 32.
//  2. CWvsContext::SendConsumeCashItemUseRequest @0xaef2f5 dispatches through
//     a jump table at @0xaef3a8, and IDA's own default-arm annotation on the
//     bound check at @0xaef3a2 reads verbatim: "ja def_AEF3A8; jumptable
//     00AEF3A8 default case, cases 34,35,37,40,45,46,53,54,62,65-68". 32 is
//     not in that default set, so type 32 has a real, non-default arm — the
//     JMS client actually sends this op.
//
// docs/packets/dispatchers/parcel.yaml:67 gives jms_v185: 10 for OPEN and
// :85 gives jms_v185: 27 for OPEN_QUICK, so PARCEL[OPEN_QUICK] renders on
// jms_v185 too.
func quickDeliveryEnabled(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(72)) ||
		(t.IsRegion("JMS") && t.MajorAtLeast(185))
}

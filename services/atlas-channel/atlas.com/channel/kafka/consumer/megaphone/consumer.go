package megaphone

import (
	consumer2 "atlas-channel/kafka/consumer"
	megaphone2 "atlas-channel/kafka/message/megaphone"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	socketmodel "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Tier and Scope values mirror Task 12's channel-side wire-up
// (socket/handler/character_cash_item_use_megaphone.go) and Task 10's
// saga-orchestrator payload; kept identical here since this consumer
// switches on the same wire strings.
const (
	tierMegaphone = "MEGAPHONE"
	tierSuper     = "SUPER"
	tierItem      = "ITEM"
	tierTriple    = "TRIPLE"

	scopeChannel = "CHANNEL"
	scopeWorld   = "WORLD"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("megaphone_broadcast")(megaphone2.EnvEventTopicMegaphone)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(megaphone2.EnvEventTopicMegaphone)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBroadcast(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleBroadcast(sc server.Model, wp writer.Producer) message.Handler[megaphone2.BroadcastEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e megaphone2.BroadcastEvent) {
		t := tenant.MustFromContext(ctx)

		// CHANNEL scope (basic megaphone): only the sender's channel renders.
		// Any other scope (WORLD) renders on every channel in the world.
		if e.Scope == scopeChannel {
			if !sc.Is(t, world.Id(e.WorldId), channel.Id(e.ChannelId)) {
				return
			}
		} else if !sc.IsWorld(t, world.Id(e.WorldId)) {
			return
		}

		if len(e.Messages) == 0 {
			l.Warnf("Megaphone broadcast event for tier [%s] has no messages; skipping.", e.Tier)
			return
		}

		var body packet.Encode
		switch e.Tier {
		case tierMegaphone:
			body = writer.WorldMessageMegaphoneBody(e.SenderMedal, e.SenderName, e.Messages[0])
		case tierSuper:
			body = writer.WorldMessageSuperMegaphoneBody(e.SenderMedal, e.SenderName, e.Messages[0], channel.Id(e.ChannelId), e.WhispersOn)
		case tierItem:
			var item *packetmodel.Asset
			if e.Item != nil {
				rebuilt := socketmodel.NewAssetFromSnapshot(*e.Item)
				item = &rebuilt
			}
			body = writer.WorldMessageItemMegaphoneBody(e.SenderMedal, e.SenderName, e.Messages[0], channel.Id(e.ChannelId), e.WhispersOn, item)
		case tierTriple:
			body = writer.WorldMessageMultiMegaphoneBody(e.SenderMedal, e.SenderName, e.Messages, channel.Id(e.ChannelId), e.WhispersOn)
		default:
			l.Warnf("Unhandled megaphone tier [%s].", e.Tier)
			return
		}

		l.WithFields(logrus.Fields{
			"character_id": e.CharacterId, "tier": e.Tier, "world_id": e.WorldId, "channel_id": e.ChannelId,
		}).Infof("Broadcasting megaphone message.")

		sessions, err := session.NewProcessor(l, ctx).AllInChannelProvider(sc.WorldId(), sc.ChannelId())
		if err != nil {
			l.WithError(err).Error("Unable to get sessions for megaphone broadcast.")
			return
		}

		announceOp := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(body)
		for _, sess := range sessions {
			if err := announceOp(sess); err != nil {
				l.WithError(err).Warnf("Unable to send megaphone broadcast to session.")
			}
		}
	}
}

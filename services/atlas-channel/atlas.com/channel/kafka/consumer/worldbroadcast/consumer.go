package worldbroadcast

import (
	consumer2 "atlas-channel/kafka/consumer"
	wb2 "atlas-channel/kafka/message/worldbroadcast"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	socketmodel "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"atlas-channel/worldbroadcast"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tvpkg "github.com/Chronicle20/atlas/libs/atlas-packet/tv"
	tvpkt "github.com/Chronicle20/atlas/libs/atlas-packet/tv/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// InitConsumers registers the world-broadcast status consumer. Like the
// megaphone broadcast consumer (Task 13), this is fire-and-forget rendering
// of a live status fanned out from atlas-world's broadcast coordinator, not a
// replayable command stream, so it uses kafka.LastOffset rather than the
// FirstOffset used by Task 9's command-side consumer.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("world_broadcast_status")(wb2.EnvEventTopicWorldBroadcastStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(wb2.EnvEventTopicWorldBroadcastStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatus(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// handleStatus renders the SEND_TV / SET_AVATAR_MEGAPHONE / clear packets
// implied by a world-broadcast STATUS event. Status events fan out to every
// channel in the world (unlike the megaphone broadcast consumer's CHANNEL
// scope), so the gate here is always sc.IsWorld, never sc.Is.
func handleStatus(sc server.Model, wp writer.Producer) message.Handler[wb2.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e wb2.StatusEvent) {
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, world.Id(e.WorldId)) {
			return
		}

		switch e.Type {
		case wb2.StatusTypeQueued:
			// Success ack to the sender only, and only for TV family: the
			// avatar megaphone family shows nothing to the client while
			// queued (design D6 / A1 delta). TvSendMessageResultSuccessBody
			// needs no config resolution — success is the bare 00 byte.
			if e.Family == worldbroadcast.FamilyTV {
				_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId,
					session.Announce(l)(ctx)(wp)(tvpkt.TvSendMessageResultWriter)(tvpkg.TvSendMessageResultSuccessBody()))
			}
		case wb2.StatusTypeStarted:
			announceStarted(l, ctx, sc, wp, e)
		case wb2.StatusTypeEnded:
			announceEnded(l, ctx, sc, wp, e)
		default:
			l.Warnf("Unhandled world broadcast status [%s].", e.Type)
		}
	}
}

// announceStarted builds the SEND_TV or SET_AVATAR_MEGAPHONE packet once and
// announces it to every session on the pod's channel (AllInChannelProvider),
// mirroring the megaphone broadcast consumer's synchronous fan-out loop.
func announceStarted(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, e wb2.StatusEvent) {
	t := tenant.MustFromContext(ctx)
	fields := logrus.Fields{
		"tenant_id": t.Id(), "world_id": e.WorldId, "family": e.Family, "character_id": e.CharacterId,
	}

	var body packet.Encode
	var writerName string
	switch e.Family {
	case worldbroadcast.FamilyTV:
		var receiverLook *packetmodel.Avatar
		if e.ReceiverLook != nil {
			rl := socketmodel.NewAvatarFromSnapshot(*e.ReceiverLook, false)
			receiverLook = &rl
		}
		// A1 delta: e.TvMessageType is the SEMANTIC key (NORMAL|STAR|HEART),
		// never a client wire byte. TvSetMessageBody resolves the wire byte
		// from the tenant messageTypes writer table (DOM-25(c)).
		body = tvpkg.TvSetMessageBody(
			tvpkg.TvMessageType(e.TvMessageType),
			socketmodel.NewAvatarFromSnapshot(e.SenderLook, false),
			writer.DecorateNameForMessage(e.SenderMedal, e.SenderName),
			e.ReceiverName,
			linesArray5(e.Messages),
			e.TotalWaitSeconds,
			receiverLook,
		)
		writerName = tvpkt.TvSetMessageWriter
		l.WithFields(fields).Infof("Rendering Maple TV message.")
	case worldbroadcast.FamilyAvatar:
		body = chatpkt.NewSetAvatarMegaphone(
			e.ItemId,
			writer.DecorateNameForMessage(e.SenderMedal, e.SenderName),
			linesArray4(e.Messages),
			uint32(e.ChannelId),
			e.WhispersOn,
			socketmodel.NewAvatarFromSnapshot(e.SenderLook, true),
		).Encode
		writerName = chatpkt.SetAvatarMegaphoneWriter
		l.WithFields(fields).Infof("Rendering avatar megaphone message.")
	default:
		l.Warnf("Unhandled world broadcast family [%s] for STARTED status.", e.Family)
		return
	}

	announceToChannel(l, ctx, sc, wp, writerName, body)
}

// announceEnded clears the TV or avatar megaphone UI for every session on the
// pod's channel. Idempotent client-side — belt-and-braces on the 10s avatar
// auto-clear (design D6).
func announceEnded(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, e wb2.StatusEvent) {
	t := tenant.MustFromContext(ctx)
	fields := logrus.Fields{
		"tenant_id": t.Id(), "world_id": e.WorldId, "family": e.Family, "character_id": e.CharacterId,
	}

	var body packet.Encode
	var writerName string
	switch e.Family {
	case worldbroadcast.FamilyTV:
		body = tvpkg.TvClearMessageBody()
		writerName = tvpkt.TvClearMessageWriter
		l.WithFields(fields).Infof("Clearing Maple TV message.")
	case worldbroadcast.FamilyAvatar:
		body = chatpkt.NewClearAvatarMegaphone().Encode
		writerName = chatpkt.ClearAvatarMegaphoneWriter
		l.WithFields(fields).Infof("Clearing avatar megaphone message.")
	default:
		l.Warnf("Unhandled world broadcast family [%s] for ENDED status.", e.Family)
		return
	}

	announceToChannel(l, ctx, sc, wp, writerName, body)
}

func announceToChannel(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, writerName string, body packet.Encode) {
	sessions, err := session.NewProcessor(l, ctx).AllInChannelProvider(sc.WorldId(), sc.ChannelId())
	if err != nil {
		l.WithError(err).Error("Unable to get sessions for world broadcast status announce.")
		return
	}

	announceOp := session.Announce(l)(ctx)(wp)(writerName)(body)
	for _, sess := range sessions {
		if err := announceOp(sess); err != nil {
			l.WithError(err).Warnf("Unable to announce world broadcast status to session.")
		}
	}
}

// linesArray5 pads or truncates messages into the fixed [5]string the TV
// packet wire requires (TvSetMessage.lines), never indexing past either
// slice's length.
func linesArray5(messages []string) [5]string {
	var lines [5]string
	n := len(messages)
	if n > len(lines) {
		n = len(lines)
	}
	for i := 0; i < n; i++ {
		lines[i] = messages[i]
	}
	return lines
}

// linesArray4 pads or truncates messages into the fixed [4]string the avatar
// megaphone packet wire requires (SetAvatarMegaphone.lines).
func linesArray4(messages []string) [4]string {
	var lines [4]string
	n := len(messages)
	if n > len(lines) {
		n = len(lines)
	}
	for i := 0; i < n; i++ {
		lines[i] = messages[i]
	}
	return lines
}

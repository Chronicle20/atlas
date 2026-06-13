package mount

import (
	"atlas-channel/character/buff"
	consumer2 "atlas-channel/kafka/consumer"
	mount2 "atlas-channel/kafka/message/mount"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// tooTiredMessage is the FR-6.3 notice sent to the rider when the mount reaches
// maximum tiredness (TooTired on a TICK reaching 99).
const tooTiredMessage = "Your mount grew tired! Treat it some revitalizer before riding it again!"

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mount_status_event")(mount2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(mount2.EnvStatusEventTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// tamingMobInfoBroadcaster is the channel-side broadcast seam for the
// MOUNT_STATUS -> SetTamingMobInfo translation. Held as a package-level var so
// tests can swap in a recording stub without standing up a REST mock for
// session resolution or _map.ForSessionsInMap. The default resolves the rider's
// session (ignoring the event if they are not on this channel) and broadcasts
// the mount info to ALL sessions in the rider's map so self + observers see it.
var tamingMobInfoBroadcaster = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model, characterId, level, exp, tiredness uint32, levelUp bool) {
	err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, func(s session.Model) error {
		return _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(),
			session.Announce(l)(ctx)(wp)(charcb.SetTamingMobInfoWriter)(writer.SetTamingMobInfoBody(characterId, level, exp, tiredness, levelUp)))
	})
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast SetTamingMobInfo for character [%d].", characterId)
	}
}

// tooTiredDismounter auto-dismounts a rider whose mount grew too tired (FR-6.3;
// mirrors Cosmic runTirednessSchedule's dispelSkill). It resolves the rider's
// session for the field, finds the active MONSTER_RIDING buff, and cancels it —
// which both visually dismounts the player and (via the buff-EXPIRED event)
// drops the active-mount registry entry so ticking stops.
var tooTiredDismounter = func(l logrus.FieldLogger, ctx context.Context, sc server.Model, characterId uint32) {
	err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, func(s session.Model) error {
		bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
		if err != nil {
			return err
		}
		for _, b := range bs {
			if buff.IsMount(b) {
				return buff.NewProcessor(l, ctx).Cancel(s.Field(), characterId, b.SourceId())
			}
		}
		return nil
	})
	if err != nil {
		l.WithError(err).Errorf("Unable to auto-dismount too-tired character [%d].", characterId)
	}
}

// tooTiredNoticer is the channel-side seam for the FR-6.3 too-tired notice. Sent
// to the rider only via the world-message NOTICE writer.
var tooTiredNoticer = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model, characterId uint32, message string) {
	err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId,
		session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessageNoticeBody(message)))
	if err != nil {
		l.WithError(err).Errorf("Unable to send too-tired notice to character [%d].", characterId)
	}
}

func handleStatusEvent(sc server.Model, wp writer.Producer) message.Handler[mount2.StatusEvent[mount2.StatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mount2.StatusEvent[mount2.StatusEventBody]) {
		switch e.Type {
		case mount2.StatusEventTypeSet, mount2.StatusEventTypeTick, mount2.StatusEventTypeFeed:
		default:
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		// Broadcast the standalone CWvsContext::OnSetTamingMobInfo packet to the
		// rider's map (self + observers). The writer opcode is registered per version
		// (v83/v84/v87=0x30, v92=0x31, v95=0x2F, jms=0x2D — IDA-verified); body is
		// characterId/level/exp/tiredness/levelUp. levelUp also drives the client's
		// level-up effect + sound.
		tamingMobInfoBroadcaster(l, ctx, wp, sc, e.CharacterId,
			uint32(e.Body.Level), uint32(e.Body.Exp), uint32(e.Body.Tiredness), e.Body.LevelUp)

		if e.Body.TooTired {
			// Cosmic dispels the mount skill when tiredness maxes; do the same so the
			// rider is actually dismounted, then send the notice.
			tooTiredDismounter(l, ctx, sc, e.CharacterId)
			tooTiredNoticer(l, ctx, wp, sc, e.CharacterId, tooTiredMessage)
		}
	}
}

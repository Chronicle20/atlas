package event

import (
	consumer2 "atlas-channel/kafka/consumer"
	event2 "atlas-channel/kafka/message/event"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("event_visual_event")(event2.EnvEventTopicEventVisual)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
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
				t, err = topic.EnvProvider(l)(event2.EnvEventTopicEventVisual)()
				if err != nil {
					return nil, err
				}
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleVisualShow(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleVisualHide(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// contiMoveBroadcaster is the channel-side broadcast seam for the CONTI_MOVE
// wire effect shared by SHOW and HIDE. Held as a package-level var so tests
// can swap in a recording stub without standing up a REST mock for
// _map.ForSessionsInMap. The default preserves the production behaviour of
// announcing through wp + session.Announce via _map.ForSessionsInMap.
//
// key selects SHOW vs HIDE; the actual state/subState wire bytes are
// resolved from the tenant's ContiMove writer options table inside
// writer.ContiMoveBody (DOM-25) -- atlas-events names the visual and
// whether it is being shown or hidden, it does not carry the wire bytes.
var contiMoveBroadcaster = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, key writer.ContiMoveKey) {
	err := _map.NewProcessor(l, ctx).ForSessionsInMap(f,
		session.Announce(l)(ctx)(wp)(fieldcb.ContiMoveWriter)(writer.ContiMoveBody(key)))
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast event visual to map [%d].", f.MapId())
	}
}

// backgroundMusicBroadcaster is the channel-side broadcast seam for the
// background-music wire effect a SHOW event may carry. Held as a
// package-level var for the same testing reason as contiMoveBroadcaster.
var backgroundMusicBroadcaster = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, bgm string) {
	err := _map.NewProcessor(l, ctx).ForSessionsInMap(f,
		session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectBackgroundMusicBody(bgm)))
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast event music to map [%d].", f.MapId())
	}
}

// handleVisualShow renders an event's visual for everyone currently in the
// map. atlas-events named the visual and its gameplay bytes; this consumer's
// whole job is to map that onto writers the channel already has registered --
// it makes no decision about whether the visual should be shown.
func handleVisualShow(sc server.Model, wp writer.Producer) message.Handler[event2.VisualEvent[event2.ShowVisualBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e event2.VisualEvent[event2.ShowVisualBody]) {
		if e.Type != event2.VisualTypeShow {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		if e.Body.Visual != event2.VisualContiMove {
			l.Warnf("Unknown visual [%s] for occurrence [%s]; ignoring.", e.Body.Visual, e.OccurrenceId)
			return
		}

		f := sc.Field(e.MapId, uuid.Nil)
		contiMoveBroadcaster(l, ctx, wp, f, writer.ContiMoveShow)

		if e.Body.Bgm == "" {
			return
		}
		backgroundMusicBroadcaster(l, ctx, wp, f, e.Body.Bgm)
	}
}

// handleVisualHide hides a previously-shown event visual. HideVisualBody
// carries no BGM field -- the elimination/arrival cleanup hides the visual
// and leaves the music playing.
func handleVisualHide(sc server.Model, wp writer.Producer) message.Handler[event2.VisualEvent[event2.HideVisualBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e event2.VisualEvent[event2.HideVisualBody]) {
		if e.Type != event2.VisualTypeHide {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		if e.Body.Visual != event2.VisualContiMove {
			l.Warnf("Unknown visual [%s] for occurrence [%s]; ignoring.", e.Body.Visual, e.OccurrenceId)
			return
		}

		f := sc.Field(e.MapId, uuid.Nil)
		contiMoveBroadcaster(l, ctx, wp, f, writer.ContiMoveHide)
	}
}

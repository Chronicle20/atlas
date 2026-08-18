package character

import (
	consumer2 "atlas-dragons/kafka/consumer"
	"context"

	dragonstate "atlas-dragons/dragon"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLogin))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLogout))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMapChanged))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChannelChanged))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleJobChanged))); err != nil {
			return err
		}
		return nil
	}
}

// handleJobChanged owns the DESTROY half of a job change: a character that left
// the dragon-bearing range loses its dragon. It needs no field — the registry
// already holds one — which is why this half lives here while the CREATE half
// lives channel-side (see the ownership table in the task-9 brief).
//
// A stage-up within the range (2210 -> 2211) resolves to HasDragon == true and
// is left alone; the channel-side CREATE refreshes the stored jobId.
//
// What the player sees: the dragon stops moving and stops generating traffic
// immediately, but the already-rendered dragon persists on every client in the
// field until the owner next leaves it. The client has no REMOVE_DRAGON handler
// arm; the only client-side teardown is destroying the owner's CUser. This is a
// client limitation, expected, and not a bug to chase.
func handleJobChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[JobChangedBody]) {
	if e.Type != StatusEventTypeJobChanged {
		return
	}
	if dragonstate.HasDragon(tenant.MustFromContext(ctx), e.Body.JobId) {
		return
	}
	if err := dragonstate.NewProcessor(l, ctx).Destroy(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon on job change for character [%d].", e.CharacterId)
	}
}

// handleLogin creates the dragon in the field the character logged into. Create
// is job-gated internally, so a non-Evan login is a cheap no-op.
func handleLogin(l logrus.FieldLogger, ctx context.Context, e StatusEvent[LoginBody]) {
	if e.Type != StatusEventTypeLogin {
		return
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	if err := dragonstate.NewProcessor(l, ctx).Create(f, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to create dragon on login for character [%d].", e.CharacterId)
	}
}

// handleLogout destroys the dragon: no dragon may outlive its owner's presence
// in a field (FR-1.5).
func handleLogout(l logrus.FieldLogger, ctx context.Context, e StatusEvent[LogoutBody]) {
	if e.Type != StatusEventTypeLogout {
		return
	}
	if err := dragonstate.NewProcessor(l, ctx).Destroy(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon on logout for character [%d].", e.CharacterId)
	}
}

// handleMapChanged destroys the dragon in the old field and recreates it in the
// new one. Destroy-then-create rather than a field update, so the old map gets
// its DESTROYED broadcast and the new map gets a SPAWN_DRAGON — exactly one of
// each, no orphan.
func handleMapChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[MapChangedBody]) {
	if e.Type != StatusEventTypeMapChanged {
		return
	}
	p := dragonstate.NewProcessor(l, ctx)
	if err := p.Destroy(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon on map change for character [%d].", e.CharacterId)
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.TargetMapId).SetInstance(e.Body.TargetInstance).Build()
	if err := p.Create(f, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to create dragon on map change for character [%d].", e.CharacterId)
	}
}

// handleChannelChanged mirrors handleMapChanged across channels.
func handleChannelChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[ChannelChangedBody]) {
	if e.Type != StatusEventTypeChannelChanged {
		return
	}
	p := dragonstate.NewProcessor(l, ctx)
	if err := p.Destroy(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon on channel change for character [%d].", e.CharacterId)
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	if err := p.Create(f, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to create dragon on channel change for character [%d].", e.CharacterId)
	}
}

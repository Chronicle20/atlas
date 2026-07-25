package character

import (
	consumer2 "atlas-mounts/kafka/consumer"
	charmsg "atlas-mounts/kafka/message/character"
	"atlas-mounts/mount"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Function seam. Production wires this to the real active-mount registry; tests
// override it with a fake to exercise the branching logic without Redis. Same
// style as the Task-18 buff consumer.
var registryRemove = func(ctx context.Context, characterId uint32) error {
	return mount.GetRegistry().Remove(ctx, characterId)
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(charmsg.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(charmsg.EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEvent))); err != nil {
			return err
		}
		return nil
	}
}

// handleStatusEvent reacts to character login/logout.
//
// On LOGOUT we drop the character's active-mount registry entry so a
// logged-out rider's tamed mount stops ticking (FR-4.4). Remove on a missing
// key is a safe no-op, which covers characters who were not riding anything.
//
// LOGIN is an intentional no-op: mounts are NOT auto-reapplied on login. The
// active-mount registry is populated when the player actually mounts, via the
// buff-APPLIED consumer (Task 18). There is no separate "online" registry to
// maintain here — the active-mount registry already contains only online
// riders, so login requires no registry action.
func handleStatusEvent(l logrus.FieldLogger, ctx context.Context, e charmsg.StatusEvent[any]) {
	if e.Type != charmsg.StatusEventTypeLogout {
		return
	}

	l.Debugf("Character [%d] logged out; clearing any active mount.", e.CharacterId)
	if err := registryRemove(ctx, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to remove active mount for character [%d] on logout.", e.CharacterId)
	}
}

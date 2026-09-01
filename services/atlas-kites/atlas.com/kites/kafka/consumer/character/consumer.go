package character

import (
	"atlas-kites/character"
	consumer2 "atlas-kites/kafka/consumer"
	character2 "atlas-kites/kafka/message/character"
	kiteMsg "atlas-kites/kafka/message/kite"
	"atlas-kites/kite"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		var err error
		t, err = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		if err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLoginBody]) {
	if e.Type != character2.EventCharacterStatusTypeLogin {
		return
	}
	// SetInstance is the difference from the chalkboards consumer this is
	// modelled on: it builds its key without the instance while its resource
	// reads with one, so instanced maps never replay. The status events have
	// always carried the instance.
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	character.NewProcessor(l, ctx).Enter(f, e.CharacterId)
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
	if e.Type != character2.EventCharacterStatusTypeLogout {
		return
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	character.NewProcessor(l, ctx).Exit(f, e.CharacterId)
	if _, err := kite.NewProcessor(l, ctx).DestroyAndEmit(e.CharacterId, kiteMsg.DestroyReasonOwnerLoggedOut); err != nil {
		l.WithError(err).Debugf("Unable to destroy kite for logged-out character [%d].", e.CharacterId)
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
	if e.Type != character2.EventCharacterStatusTypeMapChanged {
		return
	}
	// `of` is captured BEFORE the index transition purely so the
	// character-in-field index update reflects the departing map; it has no
	// bearing on where DESTROYED fans out. DestroyAndEmit takes no field at
	// all -- Destroy reads the kite's own field off the stored Model, so the
	// event is keyed and fanned out on the map the kite was actually placed
	// in regardless of this handler's ordering.
	of := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.OldMapId).SetInstance(e.Body.OldInstance).Build()
	nf := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.TargetMapId).SetInstance(e.Body.TargetInstance).Build()
	character.NewProcessor(l, ctx).TransitionMap(of, nf, e.CharacterId)
	if _, err := kite.NewProcessor(l, ctx).DestroyAndEmit(e.CharacterId, kiteMsg.DestroyReasonOwnerLeft); err != nil {
		l.WithError(err).Debugf("Unable to destroy kite for departing character [%d].", e.CharacterId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ChangeChannelEventLoginBody]) {
	if e.Type != character2.EventCharacterStatusTypeChannelChanged {
		return
	}
	of := field.NewBuilder(e.WorldId, e.Body.OldChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	nf := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	character.NewProcessor(l, ctx).TransitionChannel(of, nf, e.CharacterId)
	if _, err := kite.NewProcessor(l, ctx).DestroyAndEmit(e.CharacterId, kiteMsg.DestroyReasonOwnerLeft); err != nil {
		l.WithError(err).Debugf("Unable to destroy kite for channel-changing character [%d].", e.CharacterId)
	}
}

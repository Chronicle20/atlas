package _map

import (
	consumer2 "atlas-monsters/kafka/consumer"
	_map "atlas-monsters/map"
	"atlas-monsters/monster"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_status_event")(EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventTopicMapStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventCharacterEnter(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterEnter]) {
	if e.Type != EventTopicMapStatusTypeCharacterEnter {
		return
	}

	l.Debugf("[control-debug] CharacterEnter received: char=[%d] world=[%d] channel=[%d] map=[%d]; reassigning uncontrolled mobs.", e.Body.CharacterId, e.WorldId, e.ChannelId, e.MapId)
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()

	p := monster.NewProcessor(l, ctx)
	provider := p.NotControlledInFieldProvider(f)
	_ = model.ForEachSlice(provider, p.FindNextController(_map.CharacterIdsInFieldProvider(l)(ctx)(f)), model.ParallelExecute())
}

func handleStatusEventCharacterExit(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterExit]) {
	if e.Type != EventTopicMapStatusTypeCharacterExit {
		return
	}

	l.Debugf("[control-debug] CharacterExit received: char=[%d] world=[%d] channel=[%d] map=[%d]; releasing their controlled mobs.", e.Body.CharacterId, e.WorldId, e.ChannelId, e.MapId)
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()

	ocids, err := _map.CharacterIdsInFieldProvider(l)(ctx)(f)()
	if err != nil {
		l.WithError(err).Warnf("[control-debug] CharacterExit: unable to fetch other char ids in field for reassignment.")
		return
	}

	p := monster.NewProcessor(l, ctx)
	// Materialize the controlled-by-character list ONCE: the provider re-evaluates
	// against live registry state on each call, and StopControl mutates that
	// state. If we passed the same Provider to both ForEachSlice calls the
	// second invocation would observe the post-StopControl state (zero mobs
	// controlled by this character) and FindNextController would no-op —
	// leaving every released mob uncontrolled until something else triggers
	// reassignment. Snapshot the list first so both ops iterate the same set.
	mobs, err := p.ControlledByCharacterInFieldProvider(f, e.Body.CharacterId)()
	if err != nil {
		l.WithError(err).Warnf("[control-debug] CharacterExit: unable to fetch mobs controlled by char [%d] in field; skipping reassignment.", e.Body.CharacterId)
		return
	}
	snapshot := model.FixedProvider(mobs)
	_ = model.ForEachSlice(snapshot, p.StopControl, model.ParallelExecute())
	if len(ocids) > 0 {
		_ = model.ForEachSlice(snapshot, p.FindNextController(model.FixedProvider(ocids)), model.ParallelExecute())
	}
}

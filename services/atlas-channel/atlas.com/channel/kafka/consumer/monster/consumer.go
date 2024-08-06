package monster

import (
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameStatus)(EnvEventTopicMonsterStatus)(groupId)
	}
}

func StatusEventCreatedRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicMonsterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(sc, wp)))
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventCreatedBody]] {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventCreatedBody]) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventMonsterStatusCreated {
			return
		}

		m, err := monster.GetById(l, span, event.Tenant)(event.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being spawned.", event.UniqueId)
			return
		}

		_map.ForSessionsInMap(l, span, event.Tenant)(event.WorldId, event.ChannelId, event.MapId, spawnMonsterForSession(l, wp)(m))
	}
}

func spawnMonsterForSession(l logrus.FieldLogger, wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	spawnMonsterFunc := session.Announce(l)(wp)(writer.SpawnMonster)
	return func(m monster.Model) model.Operator[session.Model] {
		return func(s session.Model) error {
			err := spawnMonsterFunc(s, writer.SpawnMonsterBody(l, s.Tenant())(m, false))
			if err != nil {
				l.WithError(err).Errorf("Unable to spawn monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
			}
			return err
		}
	}
}

func StatusEventDestroyedRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicMonsterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(sc, wp)))
	}
}

func handleStatusEventDestroyed(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventDestroyedBody]] {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventDestroyedBody]) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventMonsterStatusDestroyed {
			return
		}

		m, err := monster.GetById(l, span, event.Tenant)(event.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being destroyed.", event.UniqueId)
			return
		}

		_map.ForSessionsInMap(l, span, event.Tenant)(event.WorldId, event.ChannelId, event.MapId, destroyMonsterForSession(l, wp)(m))
	}
}

func destroyMonsterForSession(l logrus.FieldLogger, wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	destroyMonsterFunc := session.Announce(l)(wp)(writer.DestroyMonster)
	return func(m monster.Model) model.Operator[session.Model] {
		return func(s session.Model) error {
			err := destroyMonsterFunc(s, writer.DestroyMonsterBody(l, s.Tenant())(m, writer.DestroyMonsterTypeDissapear))
			if err != nil {
				l.WithError(err).Errorf("Unable to destroy monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
			}
			return err
		}
	}
}

func StatusEventKilledRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicMonsterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventKilled(sc, wp)))
	}
}

func handleStatusEventKilled(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventKilledBody]] {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventKilledBody]) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventMonsterStatusKilled {
			return
		}

		m, err := monster.GetById(l, span, event.Tenant)(event.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being destroyed.", event.UniqueId)
			return
		}

		_map.ForSessionsInMap(l, span, event.Tenant)(event.WorldId, event.ChannelId, event.MapId, killMonsterForSession(l, wp)(m))
	}
}

func killMonsterForSession(l logrus.FieldLogger, wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	destroyMonsterFunc := session.Announce(l)(wp)(writer.DestroyMonster)
	return func(m monster.Model) model.Operator[session.Model] {
		return func(s session.Model) error {
			err := destroyMonsterFunc(s, writer.DestroyMonsterBody(l, s.Tenant())(m, writer.DestroyMonsterTypeFadeOut))
			if err != nil {
				l.WithError(err).Errorf("Unable to kill monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
			}
			return err
		}
	}
}

func StatusEventStartControlRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicMonsterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStartControl(sc, wp)))
	}
}

func handleStatusEventStartControl(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventStartControlBody]] {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventStartControlBody]) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventMonsterStatusStartControl {
			return
		}

		m, err := monster.GetById(l, span, event.Tenant)(event.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being controlled.", event.UniqueId)
			return
		}

		session.IfPresentByCharacterId(event.Tenant)(event.Body.ActorId, startControlMonsterForSession(l, wp)(m))
	}
}

func startControlMonsterForSession(l logrus.FieldLogger, wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	controlMonsterFunc := session.Announce(l)(wp)(writer.ControlMonster)
	return func(m monster.Model) model.Operator[session.Model] {
		return func(s session.Model) error {
			l.Debugf("Starting control of [%d] monster [%d] for character [%d].", m.MonsterId(), m.UniqueId(), s.CharacterId())
			err := controlMonsterFunc(s, writer.StartControlMonsterBody(l, s.Tenant())(m, false))
			if err != nil {
				l.WithError(err).Errorf("Unable to control monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
			}
			return err
		}
	}
}

func StatusEventStopControlRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicMonsterStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStopControl(sc, wp)))
	}
}

func handleStatusEventStopControl(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventStopControlBody]] {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventStopControlBody]) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventMonsterStatusStopControl {
			return
		}

		m, err := monster.GetById(l, span, event.Tenant)(event.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being controlled.", event.UniqueId)
			return
		}

		session.IfPresentByCharacterId(event.Tenant)(event.Body.ActorId, stopControlMonsterForSession(l, wp)(m))
	}
}

func stopControlMonsterForSession(l logrus.FieldLogger, wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	controlMonsterFunc := session.Announce(l)(wp)(writer.ControlMonster)
	return func(m monster.Model) model.Operator[session.Model] {
		return func(s session.Model) error {
			l.Debugf("Stopping control of [%d] monster [%d] for character [%d].", m.MonsterId(), m.UniqueId(), s.CharacterId())
			err := controlMonsterFunc(s, writer.StopControlMonsterBody(l, s.Tenant())(m))
			if err != nil {
				l.WithError(err).Errorf("Unable to control monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
			}
			return err
		}
	}
}

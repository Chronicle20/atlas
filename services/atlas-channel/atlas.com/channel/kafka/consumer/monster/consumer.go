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
			err := spawnMonsterFunc(s, writer.SpawnMonsterBody(l)(m, false))
			if err != nil {
				l.WithError(err).Errorf("Unable to spawn monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
			}
			return err
		}
	}
}

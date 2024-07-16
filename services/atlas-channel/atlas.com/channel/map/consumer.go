package _map

import (
	"atlas-channel/kafka"
	"atlas-channel/npc"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "status_event"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerStatusEvent)(EnvEventTopicMapStatus)(groupId)
	}
}

func StatusEventCharacterEnterRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		return kafka.LookupTopic(l)(EnvEventTopicMapStatus), message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter(sc, wp)))
	}
}

func StatusEventCharacterExitRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		return kafka.LookupTopic(l)(EnvEventTopicMapStatus), message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit(sc, wp)))
	}
}

func handleStatusEventCharacterEnter(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[characterEnter]) {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[characterEnter]) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventTopicMapStatusTypeCharacterEnter {
			return
		}

		l.Debugf("Received MapStatus [%s] Event. characterId [%d] worldId [%d] channelId [%d] mapId [%d].", event.Type, event.Body.CharacterId, event.WorldId, event.ChannelId, event.MapId)
		session.IfPresentByCharacterId(event.Tenant)(event.Body.CharacterId, enterMap(l, span, event.Tenant, wp)(event.MapId))
	}
}

func enterMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model, wp writer.Producer) func(mapId uint32) model.Operator[session.Model] {
	return func(mapId uint32) model.Operator[session.Model] {
		return func(s session.Model) error {
			l.Debugf("Processing character [%d] entering map [%d].", s.CharacterId(), mapId)
			// retrieve character ids in map
			// spawn new character for others
			// spawn other characters for incoming

			go npc.ForEachInMap(l, span, tenant)(mapId, spawnNPCForSession(l, wp)(s))

			// fetch monsters in map

			// fetch drops in map

			// fetch reactors in map
			return nil
		}
	}
}

func handleStatusEventCharacterExit(sc server.Model, _ writer.Producer) func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[characterExit]) {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[characterExit]) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
			return
		}

		if event.Type == EventTopicMapStatusTypeCharacterExit {
			l.Debugf("Received MapStatus [%s] Event. characterId [%d] worldId [%d] channelId [%d] mapId [%d].", event.Type, event.Body.CharacterId, event.WorldId, event.ChannelId, event.MapId)
			return
		}
	}
}

func spawnNPCForSession(l logrus.FieldLogger, wp writer.Producer) func(s session.Model) model.Operator[npc.Model] {
	spawnNPCFunc := session.Announce(wp)(writer.SpawnNPC)
	spawnNPCRequestControllerFunc := session.Announce(wp)(writer.SpawnNPCRequestController)
	return func(s session.Model) model.Operator[npc.Model] {
		return func(n npc.Model) error {
			err := spawnNPCFunc(s, writer.SpawnNPCBody(l)(n))
			if err != nil {
				l.WithError(err).Errorf("Unable to spawn npc [%d] for character [%d].", n.Template(), s.CharacterId())
				return err
			}
			err = spawnNPCRequestControllerFunc(s, writer.SpawnNPCRequestControllerBody(l)(n, true))
			if err != nil {
				l.WithError(err).Errorf("Unable to spawn npc [%d] for character [%d].", n.Template(), s.CharacterId())
			}
			return err
		}
	}
}

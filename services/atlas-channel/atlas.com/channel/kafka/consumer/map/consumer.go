package _map

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/npc"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "status_event"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvEventTopicMapStatus)(groupId)
	}
}

func StatusEventCharacterEnterRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicMapStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter(sc, wp)))
	}
}

func StatusEventCharacterExitRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvEventTopicMapStatus)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit(sc, wp)))
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

type KeyProvider[M any, K comparable] func(m M) K

type ValueProvider[M any, V any] func(m M) V

func MapProvider[M any, K comparable, V any](mp model.Provider[[]M], kp KeyProvider[M, K], vp ValueProvider[M, V]) model.Provider[map[K]V] {
	ms, err := mp()
	if err != nil {
		return model.ErrorProvider[map[K]V](err)
	}
	return func() (map[K]V, error) {
		var result = make(map[K]V)
		for _, m := range ms {
			result[kp(m)] = vp(m)
		}
		return result, nil
	}
}

func enterMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model, wp writer.Producer) func(mapId uint32) model.Operator[session.Model] {
	return func(mapId uint32) model.Operator[session.Model] {
		return func(s session.Model) error {
			l.Debugf("Processing character [%d] entering map [%d].", s.CharacterId(), mapId)
			ids, err := _map.GetCharacterIdsInMap(l, span, s.Tenant())(s.WorldId(), s.ChannelId(), mapId)
			if err != nil {
				l.WithError(err).Errorf("No characters found in map [%d] for world [%d] and channel [%d].", mapId, s.WorldId(), s.ChannelId())
				return err
			}

			cp := model.SliceMap(model.FixedProvider(ids), character.GetById(l, span, tenant), model.ParallelMap())
			cms, err := MapProvider[character.Model, uint32, character.Model](cp, GetId, GetModel)()
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character details for characters in map.")
				return err
			}

			// spawn new character for others
			for k := range cms {
				if k != s.CharacterId() {
					session.IfPresentByCharacterId(s.Tenant())(k, spawnCharacterForSession(l, wp)(cms[s.CharacterId()], true))
				}
			}

			// spawn other characters for incoming
			for k, v := range cms {
				if k != s.CharacterId() {
					_ = spawnCharacterForSession(l, wp)(v, false)(s)
				}
			}

			go npc.ForEachInMap(l, span, tenant)(mapId, spawnNPCForSession(l, wp)(s))

			go monster.ForEachInMap(l, span, tenant)(s.WorldId(), s.ChannelId(), mapId, spawnMonsterForSession(l, wp)(s))

			// fetch drops in map

			// fetch reactors in map
			return nil
		}
	}
}

func spawnCharacterForSession(l logrus.FieldLogger, wp writer.Producer) func(c character.Model, enteringField bool) model.Operator[session.Model] {
	spawnCharacterFunc := session.Announce(l)(wp)(writer.CharacterSpawn)
	return func(c character.Model, enteringField bool) model.Operator[session.Model] {
		return func(s session.Model) error {
			err := spawnCharacterFunc(s, writer.CharacterSpawnBody(l, s.Tenant())(c, enteringField))
			if err != nil {
				l.WithError(err).Errorf("Unable to spawn character [%d] for [%d]", c.Id(), s.CharacterId())
				return err
			}
			return nil
		}
	}
}

func GetModel(m character.Model) character.Model {
	return m
}

func GetId(m character.Model) uint32 {
	return m.Id()
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
	spawnNPCFunc := session.Announce(l)(wp)(writer.SpawnNPC)
	spawnNPCRequestControllerFunc := session.Announce(l)(wp)(writer.SpawnNPCRequestController)
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

func spawnMonsterForSession(l logrus.FieldLogger, wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
	spawnMonsterFunc := session.Announce(l)(wp)(writer.SpawnMonster)
	return func(s session.Model) model.Operator[monster.Model] {
		return func(m monster.Model) error {
			l.Debugf("Spawning [%d] monster [%d] for character [%d].", m.MonsterId(), m.UniqueId(), s.CharacterId())
			err := spawnMonsterFunc(s, writer.SpawnMonsterBody(l, s.Tenant())(m, false))
			if err != nil {
				l.WithError(err).Errorf("Unable to spawn monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
			}
			return err
		}
	}
}

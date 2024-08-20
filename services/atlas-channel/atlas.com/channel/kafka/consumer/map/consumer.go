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

		l.Debugf("Character [%d] has entered map [%d] in worldId [%d] channelId [%d].", event.Body.CharacterId, event.MapId, event.WorldId, event.ChannelId)
		session.IfPresentByCharacterId(event.Tenant, sc.WorldId(), sc.ChannelId())(event.Body.CharacterId, enterMap(l, span, event.Tenant, wp)(event.MapId))
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

			cp := model.SliceMap(model.FixedProvider(ids), character.GetByIdWithInventory(l, span, tenant), model.ParallelMap())
			cms, err := model.CollectToMap(cp, GetId, GetModel)()
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character details for characters in map.")
				return err
			}

			// spawn new character for others
			for k := range cms {
				if k != s.CharacterId() {
					session.IfPresentByCharacterId(s.Tenant(), s.WorldId(), s.ChannelId())(k, spawnCharacterForSession(l, wp)(cms[s.CharacterId()], true))
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

func handleStatusEventCharacterExit(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[characterExit]) {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[characterExit]) {
		if !sc.Is(event.Tenant, event.WorldId, event.ChannelId) {
			return
		}

		if event.Type == EventTopicMapStatusTypeCharacterExit {
			l.Debugf("Character [%d] has left map [%d] in worldId [%d] channelId [%d].", event.Body.CharacterId, event.MapId, event.WorldId, event.ChannelId)
			_ = _map.ForOtherSessionsInMap(l, span, event.Tenant)(event.WorldId, event.ChannelId, event.MapId, event.Body.CharacterId, despawnForSession(l, event.Tenant, wp)(event.Body.CharacterId))
			return
		}
	}
}

func despawnForSession(l logrus.FieldLogger, tenant tenant.Model, wp writer.Producer) func(id uint32) model.Operator[session.Model] {
	despawnCharacterFunc := session.Announce(l)(wp)(writer.CharacterDespawn)
	return func(id uint32) model.Operator[session.Model] {
		return func(s session.Model) error {
			err := despawnCharacterFunc(s, writer.CharacterDespawnBody(l, tenant)(id))
			if err != nil {
				l.WithError(err).Errorf("Unable to despawn character [%d] for character [%d].", id, s.CharacterId())
			}
			return err
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

package _map

import (
	"atlas-channel/character"
	"atlas-channel/guild"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/npc"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_status_event")(EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventTopicMapStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit(sc, wp))))
			}
		}
	}
}

func handleStatusEventCharacterEnter(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[characterEnter]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[characterEnter]) {
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		if event.Type != EventTopicMapStatusTypeCharacterEnter {
			return
		}

		l.Debugf("Character [%d] has entered map [%d] in worldId [%d] channelId [%d].", event.Body.CharacterId, event.MapId, event.WorldId, event.ChannelId)
		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(event.Body.CharacterId, enterMap(l, ctx, wp)(event.MapId))
	}
}

func enterMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(mapId uint32) model.Operator[session.Model] {
	return func(mapId uint32) model.Operator[session.Model] {
		return func(s session.Model) error {
			l.Debugf("Processing character [%d] entering map [%d].", s.CharacterId(), mapId)
			ids, err := _map.GetCharacterIdsInMap(l)(ctx)(s.WorldId(), s.ChannelId(), mapId)
			if err != nil {
				l.WithError(err).Errorf("No characters found in map [%d] for world [%d] and channel [%d].", mapId, s.WorldId(), s.ChannelId())
				return err
			}

			cp := model.SliceMap(character.GetByIdWithInventory(l)(ctx))(model.FixedProvider(ids))(model.ParallelMap())
			cms, err := model.CollectToMap(cp, GetId, GetModel)()
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character details for characters in map.")
				return err
			}
			g, err := guild.GetByMemberId(l)(ctx)(s.CharacterId())

			// spawn new character for others
			for k := range cms {
				if k != s.CharacterId() {
					session.IfPresentByCharacterId(tenant.MustFromContext(ctx), s.WorldId(), s.ChannelId())(k, spawnCharacterForSession(l)(ctx)(wp)(cms[s.CharacterId()], g, true))
				}
			}

			// spawn other characters for incoming
			for k, v := range cms {
				if k != s.CharacterId() {
					kg, _ := guild.GetByMemberId(l)(ctx)(k)
					_ = spawnCharacterForSession(l)(ctx)(wp)(v, kg, false)(s)
				}
			}

			go npc.ForEachInMap(l)(ctx)(mapId, spawnNPCForSession(l)(ctx)(wp)(s))

			go monster.ForEachInMap(l)(ctx)(s.WorldId(), s.ChannelId(), mapId, spawnMonsterForSession(l)(ctx)(wp)(s))

			// fetch drops in map

			// fetch reactors in map
			return nil
		}
	}
}

func spawnCharacterForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
			spawnCharacterFunc := session.Announce(l)(ctx)(wp)(writer.CharacterSpawn)
			return func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := spawnCharacterFunc(s, writer.CharacterSpawnBody(l, tenant.MustFromContext(ctx))(c, g, enteringField))
					if err != nil {
						l.WithError(err).Errorf("Unable to spawn character [%d] for [%d]", c.Id(), s.CharacterId())
						return err
					}
					return nil
				}
			}
		}
	}
}

func GetModel(m character.Model) character.Model {
	return m
}

func GetId(m character.Model) uint32 {
	return m.Id()
}

func handleStatusEventCharacterExit(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event statusEvent[characterExit]) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent[characterExit]) {
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.ChannelId) {
			return
		}

		if event.Type == EventTopicMapStatusTypeCharacterExit {
			l.Debugf("Character [%d] has left map [%d] in worldId [%d] channelId [%d].", event.Body.CharacterId, event.MapId, event.WorldId, event.ChannelId)
			_ = _map.ForOtherSessionsInMap(l)(ctx)(event.WorldId, event.ChannelId, event.MapId, event.Body.CharacterId, despawnForSession(l)(ctx)(wp)(event.Body.CharacterId))
			return
		}
	}
}

func despawnForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(id uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(id uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(id uint32) model.Operator[session.Model] {
			despawnCharacterFunc := session.Announce(l)(ctx)(wp)(writer.CharacterDespawn)
			return func(id uint32) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := despawnCharacterFunc(s, writer.CharacterDespawnBody(id))
					if err != nil {
						l.WithError(err).Errorf("Unable to despawn character [%d] for character [%d].", id, s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func spawnNPCForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[npc.Model] {
			spawnNPCFunc := session.Announce(l)(ctx)(wp)(writer.SpawnNPC)
			spawnNPCRequestControllerFunc := session.Announce(l)(ctx)(wp)(writer.SpawnNPCRequestController)
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
	}
}

func spawnMonsterForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
			spawnMonsterFunc := session.Announce(l)(ctx)(wp)(writer.SpawnMonster)
			return func(s session.Model) model.Operator[monster.Model] {
				return func(m monster.Model) error {
					l.Debugf("Spawning [%d] monster [%d] for character [%d].", m.MonsterId(), m.UniqueId(), s.CharacterId())
					err := spawnMonsterFunc(s, writer.SpawnMonsterBody(l, tenant.MustFromContext(ctx))(m, false))
					if err != nil {
						l.WithError(err).Errorf("Unable to spawn monster [%d] for character [%d].", m.UniqueId(), s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

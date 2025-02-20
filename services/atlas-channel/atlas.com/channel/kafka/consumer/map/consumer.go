package _map

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/drop"
	"atlas-channel/guild"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/npc"
	"atlas-channel/reactor"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterEnter]) {
		if e.Type != EventTopicMapStatusTypeCharacterEnter {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		l.Debugf("Character [%d] has entered map [%d] in worldId [%d] channelId [%d].", e.Body.CharacterId, e.MapId, e.WorldId, e.ChannelId)
		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, enterMap(l, ctx, wp)(sc.Map(_map2.Id(e.MapId))))
	}
}

func enterMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(m _map2.Model) model.Operator[session.Model] {
	return func(m _map2.Model) model.Operator[session.Model] {
		return func(s session.Model) error {
			l.Debugf("Processing character [%d] entering map [%d].", s.CharacterId(), m.MapId())
			ids, err := _map.GetCharacterIdsInMap(l)(ctx)(m)
			if err != nil {
				l.WithError(err).Errorf("No characters found in map [%d] for world [%d] and channel [%d].", m.MapId(), s.WorldId(), s.ChannelId())
				return err
			}

			cp := model.SliceMap(character.GetByIdWithInventory(l)(ctx)())(model.FixedProvider(ids))(model.ParallelMap())
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

			go npc.ForEachInMap(l)(ctx)(m.MapId(), spawnNPCForSession(l)(ctx)(wp)(s))

			go monster.ForEachInMap(l)(ctx)(m, spawnMonsterForSession(l)(ctx)(wp)(s))

			go drop.ForEachInMap(l)(ctx)(m, spawnDropsForSession(l)(ctx)(wp)(s))

			go reactor.ForEachInMap(l)(ctx)(m, spawnReactorsForSession(l)(ctx)(wp)(s))
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
					bs, err := buff.GetByCharacterId(l)(ctx)(c.Id())
					if err != nil {
						bs = make([]buff.Model, 0)
					}

					err = spawnCharacterFunc(s, writer.CharacterSpawnBody(l, tenant.MustFromContext(ctx))(c, bs, g, enteringField))
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
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[characterExit]) {
		if e.Type != EventTopicMapStatusTypeCharacterExit {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		l.Debugf("Character [%d] has left map [%d] in worldId [%d] channelId [%d].", e.Body.CharacterId, e.MapId, e.WorldId, e.ChannelId)
		_ = _map.ForOtherSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), e.Body.CharacterId, despawnForSession(l)(ctx)(wp)(e.Body.CharacterId))
		return
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

func spawnDropsForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[drop.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[drop.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[drop.Model] {
			return func(s session.Model) model.Operator[drop.Model] {
				return func(d drop.Model) error {
					l.Debugf("Spawning [%d] drop [%d] for character [%d].", d.ItemId(), d.Id(), s.CharacterId())
					err := session.Announce(l)(ctx)(wp)(writer.DropSpawn)(s, writer.DropSpawnBody(l, tenant.MustFromContext(ctx))(d, writer.DropEnterTypeExisting, 0))
					if err != nil {
						l.WithError(err).Errorf("Unable to spawn drop [%d] for character [%d].", d.Id(), s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

func spawnReactorsForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[reactor.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[reactor.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[reactor.Model] {
			return func(s session.Model) model.Operator[reactor.Model] {
				return func(r reactor.Model) error {
					l.Debugf("Spawning [%d] reactor [%d] for character [%d].", r.Classification(), r.Id(), s.CharacterId())
					err := session.Announce(l)(ctx)(wp)(writer.ReactorSpawn)(s, writer.ReactorSpawnBody(l, tenant.MustFromContext(ctx))(r))
					if err != nil {
						l.WithError(err).Errorf("Unable to spawn reactor [%d] for character [%d].", r.Id(), s.CharacterId())
					}
					return err
				}
			}
		}
	}
}

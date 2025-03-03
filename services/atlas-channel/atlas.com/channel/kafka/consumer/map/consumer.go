package _map

import (
	"atlas-channel/chair"
	"atlas-channel/chalkboard"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/drop"
	"atlas-channel/guild"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/npc"
	"atlas-channel/party"
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
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_status_event")(EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
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
					err = session.IfPresentByCharacterId(tenant.MustFromContext(ctx), s.WorldId(), s.ChannelId())(k, spawnCharacterForSession(l)(ctx)(wp)(cms[s.CharacterId()], g, true))
					if err != nil {
						l.WithError(err).Errorf("Unable to spawn character [%d] for [%d]", s.CharacterId(), k)
					}
				}
			}

			// spawn other characters for incoming
			for k, v := range cms {
				if k != s.CharacterId() {
					kg, _ := guild.GetByMemberId(l)(ctx)(k)
					err = spawnCharacterForSession(l)(ctx)(wp)(v, kg, false)(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to spawn character [%d] for [%d]", v.Id(), s.CharacterId())
					}
				}
			}

			go func() {
				err = npc.ForEachInMap(l)(ctx)(m.MapId(), spawnNPCForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn npcs for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = monster.ForEachInMap(l)(ctx)(m, spawnMonsterForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn monsters for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = drop.ForEachInMap(l)(ctx)(m, spawnDropsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn drops for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = reactor.ForEachInMap(l)(ctx)(m, spawnReactorsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn reactors for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = chalkboard.ForEachInMap(l)(ctx)(m, spawnChalkboardsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn drops for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = chair.ForEachInMap(l)(ctx)(m, spawnChairsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn drops for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				imf := party.OtherMemberInMap(s.WorldId(), s.ChannelId(), s.MapId(), s.CharacterId())
				oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(party.ByMemberIdProvider(l)(ctx)(s.CharacterId())))
				err = session.ForEachByCharacterId(s.Tenant(), s.WorldId(), s.ChannelId())(oip, session.Announce(l)(ctx)(wp)(writer.PartyMemberHP)(writer.PartyMemberHPBody(s.CharacterId(), cms[s.CharacterId()].Hp(), cms[s.CharacterId()].MaxHp())))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] health to party members.", s.CharacterId())
				}

				_ = model.ForEachSlice(oip, func(oid uint32) error {
					return session.Announce(l)(ctx)(wp)(writer.PartyMemberHP)(writer.PartyMemberHPBody(oid, cms[oid].Hp(), cms[oid].MaxHp()))(s)
				}, model.ParallelExecute())
			}()
			return nil
		}
	}
}

func spawnCharacterForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
			return func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
				return func(s session.Model) error {
					bs, err := buff.GetByCharacterId(l)(ctx)(c.Id())
					if err != nil {
						bs = make([]buff.Model, 0)
					}

					return session.Announce(l)(ctx)(wp)(writer.CharacterSpawn)(writer.CharacterSpawnBody(l, tenant.MustFromContext(ctx))(c, bs, g, enteringField))(s)
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
		err := _map.ForOtherSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), e.Body.CharacterId, despawnForSession(l)(ctx)(wp)(e.Body.CharacterId))
		if err != nil {
			l.WithError(err).Errorf("Unable to despawn character [%d] for characters in map [%d].", e.Body.CharacterId, e.MapId)
		}
		return
	}
}

func despawnForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(id uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(id uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(id uint32) model.Operator[session.Model] {
			return func(id uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterDespawn)(writer.CharacterDespawnBody(id))
			}
		}
	}
}

func spawnNPCForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[npc.Model] {
			return func(s session.Model) model.Operator[npc.Model] {
				return func(n npc.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.SpawnNPC)(writer.SpawnNPCBody(l)(n))(s)
					if err != nil {
						return err
					}
					return session.Announce(l)(ctx)(wp)(writer.SpawnNPCRequestController)(writer.SpawnNPCRequestControllerBody(l)(n, true))(s)
				}
			}
		}
	}
}

func spawnMonsterForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
			return func(s session.Model) model.Operator[monster.Model] {
				return func(m monster.Model) error {
					return session.Announce(l)(ctx)(wp)(writer.SpawnMonster)(writer.SpawnMonsterBody(l, tenant.MustFromContext(ctx))(m, false))(s)
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
					return session.Announce(l)(ctx)(wp)(writer.DropSpawn)(writer.DropSpawnBody(l, tenant.MustFromContext(ctx))(d, writer.DropEnterTypeExisting, 0))(s)
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
					return session.Announce(l)(ctx)(wp)(writer.ReactorSpawn)(writer.ReactorSpawnBody(l, tenant.MustFromContext(ctx))(r))(s)
				}
			}
		}
	}
}

func spawnChalkboardsForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[chalkboard.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[chalkboard.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[chalkboard.Model] {
			return func(s session.Model) model.Operator[chalkboard.Model] {
				return func(c chalkboard.Model) error {
					return session.Announce(l)(ctx)(wp)(writer.ChalkboardUse)(writer.ChalkboardUseBody(c.Id(), c.Message()))(s)
				}
			}
		}
	}
}

func spawnChairsForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[chair.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[chair.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[chair.Model] {
			return func(s session.Model) model.Operator[chair.Model] {
				return func(c chair.Model) error {
					return session.Announce(l)(ctx)(wp)(writer.CharacterShowChair)(writer.CharacterShowChairBody(c.CharacterId(), c.Id()))(s)
				}
			}
		}
	}
}

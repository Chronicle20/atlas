package _map

import (
	"atlas-channel/chair"
	"atlas-channel/chalkboard"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	mapData "atlas-channel/data/map"
	npc2 "atlas-channel/data/npc"
	"atlas-channel/drop"
	"atlas-channel/guild"
	consumer2 "atlas-channel/kafka/consumer"
	_map3 "atlas-channel/kafka/message/map"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/party"
	"atlas-channel/reactor"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/transport/route"
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
	"time"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_status_event")(_map3.EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(_map3.EnvEventTopicMapStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit(sc, wp))))
			}
		}
	}
}

func handleStatusEventCharacterEnter(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.CharacterEnter]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.CharacterEnter]) {
		if e.Type != _map3.EventTopicMapStatusTypeCharacterEnter {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		l.Debugf("Character [%d] has entered map [%d] in worldId [%d] channelId [%d].", e.Body.CharacterId, e.MapId, e.WorldId, e.ChannelId)
		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, enterMap(l, ctx, wp)(sc.Map(_map2.Id(e.MapId))))
	}
}

func enterMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(m _map2.Model) model.Operator[session.Model] {
	t := tenant.MustFromContext(ctx)
	return func(m _map2.Model) model.Operator[session.Model] {
		return func(s session.Model) error {
			l.Debugf("Processing character [%d] entering map [%d].", s.CharacterId(), m.MapId())
			ids, err := _map.NewProcessor(l, ctx).GetCharacterIdsInMap(m)
			if err != nil {
				l.WithError(err).Errorf("No characters found in map [%d] for world [%d] and channel [%d].", m.MapId(), s.WorldId(), s.ChannelId())
				return err
			}

			cp := character.NewProcessor(l, ctx)
			cmp := model.SliceMap(cp.GetById(cp.InventoryDecorator, cp.PetModelDecorator))(model.FixedProvider(ids))(model.ParallelMap())
			cms, err := model.CollectToMap(cmp, GetId, GetModel)()
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character details for characters in map.")
				return err
			}
			g, err := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())

			// spawn new character for others
			for k := range cms {
				if k != s.CharacterId() {
					err = session.NewProcessor(l, ctx).IfPresentByCharacterId(s.WorldId(), s.ChannelId())(k, spawnCharacterForSession(l)(ctx)(wp)(cms[s.CharacterId()], g, true))
					if err != nil {
						l.WithError(err).Errorf("Unable to spawn character [%d] for [%d]", s.CharacterId(), k)
					}
				}
			}

			// spawn other characters for incoming
			for k, v := range cms {
				if k != s.CharacterId() {
					kg, _ := guild.NewProcessor(l, ctx).GetByMemberId(k)
					err = spawnCharacterForSession(l)(ctx)(wp)(v, kg, false)(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to spawn character [%d] for [%d]", v.Id(), s.CharacterId())
					}
				}
			}

			go func() {
				for k, v := range cms {
					if k != s.CharacterId() {
						for _, p := range v.Pets() {
							if p.Slot() >= 0 {
								err = session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetSpawnBody(l)(t)(p))(s)
								if err != nil {
									l.WithError(err).Errorf("Unable to spawn character [%d] pet for [%d]", k, s.CharacterId())
								}
							}
						}
					}
				}
			}()

			go func() {
				for k := range cms {
					for _, p := range cms[s.CharacterId()].Pets() {
						if p.Slot() >= 0 {
							err = session.NewProcessor(l, ctx).IfPresentByCharacterId(s.WorldId(), s.ChannelId())(k, session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetSpawnBody(l)(t)(p)))
							if err != nil {
								l.WithError(err).Errorf("Unable to spawn character [%d] pet for [%d]", s.CharacterId(), k)
							}
							err = session.Announce(l)(ctx)(wp)(writer.PetExcludeResponse)(writer.PetExcludeResponseBody(p))(s)
							if err != nil {
								l.WithError(err).Errorf("Unable to announce pet [%d] exclusion list to character [%d].", p.Id(), s.CharacterId())
							}
						}
					}
				}
			}()

			go func() {
				err = npc2.NewProcessor(l, ctx).ForEachInMap(m.MapId(), spawnNPCForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn npcs for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = monster.NewProcessor(l, ctx).ForEachInMap(m, spawnMonsterForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn monsters for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = drop.NewProcessor(l, ctx).ForEachInMap(m, spawnDropsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn drops for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = reactor.NewProcessor(l, ctx).ForEachInMap(m, spawnReactorsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn reactors for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = chalkboard.NewProcessor(l, ctx).ForEachInMap(m, spawnChalkboardsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn drops for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = chair.NewProcessor(l, ctx).ForEachInMap(m, spawnChairsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn drops for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				imf := party.OtherMemberInMap(s.WorldId(), s.ChannelId(), s.MapId(), s.CharacterId())
				oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(party.NewProcessor(l, ctx).ByMemberIdProvider(s.CharacterId())))
				err = session.NewProcessor(l, ctx).ForEachByCharacterId(s.WorldId(), s.ChannelId())(oip, session.Announce(l)(ctx)(wp)(writer.PartyMemberHP)(writer.PartyMemberHPBody(s.CharacterId(), cms[s.CharacterId()].Hp(), cms[s.CharacterId()].MaxHp())))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] health to party members.", s.CharacterId())
				}

				_ = model.ForEachSlice(oip, func(oid uint32) error {
					return session.Announce(l)(ctx)(wp)(writer.PartyMemberHP)(writer.PartyMemberHPBody(oid, cms[oid].Hp(), cms[oid].MaxHp()))(s)
				}, model.ParallelExecute())
			}()

			go func() {
				var md mapData.Model
				md, err = mapData.NewProcessor(l, ctx).GetById(m.MapId())
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve map data for map [%d].", m.MapId())
					return
				}
				if md.Clock() {
					_ = session.Announce(l)(ctx)(wp)(writer.Clock)(writer.TownClockBody(l, t)(time.Now()))(s)
				}
			}()

			go func() {
				var hasShip bool
				hasShip, err = route.NewProcessor(l, ctx).IsBoatInMap(m.MapId())
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve boat data for map [%d].", m.MapId())
					return
				}
				if hasShip {
					_ = session.Announce(l)(ctx)(wp)(writer.FieldTransportState)(writer.FieldTransportStateBody(l, t)(writer.TransportStateEnter1, false))(s)
				} else {
					_ = session.Announce(l)(ctx)(wp)(writer.FieldTransportState)(writer.FieldTransportStateBody(l, t)(writer.TransportStateMove1, false))(s)
				}
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
					bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(c.Id())
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

func handleStatusEventCharacterExit(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.CharacterExit]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.CharacterExit]) {
		if e.Type != _map3.EventTopicMapStatusTypeCharacterExit {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		l.Debugf("Character [%d] has left map [%d] in worldId [%d] channelId [%d].", e.Body.CharacterId, e.MapId, e.WorldId, e.ChannelId)
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Map(_map2.Id(e.MapId)), e.Body.CharacterId, despawnForSession(l)(ctx)(wp)(e.Body.CharacterId))
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

func spawnNPCForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc2.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc2.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[npc2.Model] {
			return func(s session.Model) model.Operator[npc2.Model] {
				return func(n npc2.Model) error {
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

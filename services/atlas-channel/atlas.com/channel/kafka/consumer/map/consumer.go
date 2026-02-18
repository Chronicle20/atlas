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
	"atlas-channel/party_quest"
	"atlas-channel/reactor"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/transport/route"
	"atlas-channel/weather"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
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
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventWeatherStart(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventWeatherEnd(sc, wp))))
			}
		}
	}
}

func handleStatusEventCharacterEnter(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.CharacterEnter]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.CharacterEnter]) {
		if e.Type != _map3.EventTopicMapStatusTypeCharacterEnter {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Character [%d] has entered map [%d] instance [%s] in worldId [%d] channelId [%d].", e.Body.CharacterId, e.MapId, e.Instance, e.WorldId, e.ChannelId)
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, enterMap(l, ctx, wp)(f))
	}
}

func enterMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model) model.Operator[session.Model] {
	t := tenant.MustFromContext(ctx)
	return func(f field.Model) model.Operator[session.Model] {
		return func(s session.Model) error {
			l.Debugf("Processing character [%d] entering map [%d] instance [%s].", s.CharacterId(), f.MapId(), f.Instance())
			ids, err := _map.NewProcessor(l, ctx).GetCharacterIdsInMap(f)
			if err != nil {
				l.WithError(err).Errorf("No characters found in map [%d] instance [%s] for world [%d] and channel [%d].", f.MapId(), f.Instance(), s.WorldId(), s.ChannelId())
				return err
			}

			cp := character.NewProcessor(l, ctx)
			cmp := model.SliceMap(cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator, cp.PetModelDecorator))(model.FixedProvider(ids))(model.ParallelMap())
			cms, err := model.CollectToMap(cmp, GetId, GetModel)()
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character details for characters in map.")
				return err
			}
			g, err := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())

			// spawn new character for others
			for k := range cms {
				if k != s.CharacterId() {
					err = session.NewProcessor(l, ctx).IfPresentByCharacterId(s.Field().Channel())(k, spawnCharacterForSession(l)(ctx)(wp)(cms[s.CharacterId()], g, true))
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
							err = session.NewProcessor(l, ctx).IfPresentByCharacterId(s.Field().Channel())(k, session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetSpawnBody(l)(t)(p)))
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
				err = npc2.NewProcessor(l, ctx).ForEachInMap(f.MapId(), spawnNPCForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn npcs for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = monster.NewProcessor(l, ctx).ForEachInMap(f, spawnMonsterForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn monsters for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = drop.NewProcessor(l, ctx).ForEachInMap(f, spawnDropsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn drops for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = reactor.NewProcessor(l, ctx).ForEachInMap(f, spawnReactorsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn reactors for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = chalkboard.NewProcessor(l, ctx).ForEachInMap(f, spawnChalkboardsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn drops for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				err = chair.NewProcessor(l, ctx).ForEachInMap(f, spawnChairsForSession(l)(ctx)(wp)(s))
				if err != nil {
					l.WithError(err).Errorf("Unable to spawn drops for character [%d].", s.CharacterId())
				}
			}()

			go func() {
				imf := party.OtherMemberInMap(s.Field(), s.CharacterId())
				oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(party.NewProcessor(l, ctx).ByMemberIdProvider(s.CharacterId())))
				err = session.NewProcessor(l, ctx).ForEachByCharacterId(s.Field().Channel())(oip, session.Announce(l)(ctx)(wp)(writer.PartyMemberHP)(writer.PartyMemberHPBody(s.CharacterId(), cms[s.CharacterId()].Hp(), cms[s.CharacterId()].MaxHp())))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] health to party members.", s.CharacterId())
				}

				_ = model.ForEachSlice(oip, func(oid uint32) error {
					return session.Announce(l)(ctx)(wp)(writer.PartyMemberHP)(writer.PartyMemberHPBody(oid, cms[oid].Hp(), cms[oid].MaxHp()))(s)
				}, model.ParallelExecute())
			}()

			go func() {
				var md mapData.Model
				md, err = mapData.NewProcessor(l, ctx).GetById(f.MapId())
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve map data for map [%d].", f.MapId())
					return
				}
				if md.Clock() {
					_ = session.Announce(l)(ctx)(wp)(writer.Clock)(writer.TownClockBody(l, t)(time.Now()))(s)
				}
			}()

			go func() {
				var hasShip bool
				hasShip, err = route.NewProcessor(l, ctx).IsBoatInMap(f.MapId())
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve boat data for map [%d].", f.MapId())
					return
				}
				if hasShip {
					_ = session.Announce(l)(ctx)(wp)(writer.FieldTransportState)(writer.FieldTransportStateBody(l)(writer.TransportStateEnter1, false))(s)
				} else {
					_ = session.Announce(l)(ctx)(wp)(writer.FieldTransportState)(writer.FieldTransportStateBody(l)(writer.TransportStateMove1, false))(s)
				}
			}()

			go func() {
				timer, terr := party_quest.NewProcessor(l, ctx).GetTimerByCharacterId(s.CharacterId())
				if terr != nil {
					return
				}
				if timer.Duration() <= 0 {
					return
				}
				_ = session.Announce(l)(ctx)(wp)(writer.Clock)(writer.TimerClockBody(l, t)(timer.Duration()))(s)
			}()

			go func() {
				we, werr := weather.NewProcessor(l, ctx).GetActive(f)
				if werr != nil {
					return
				}
				_ = session.Announce(l)(ctx)(wp)(writer.FieldEffectWeather)(writer.FieldEffectWeatherStartBody(l)(we.ItemId, we.Message))(s)
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

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Character [%d] has left map [%d] instance [%s] in worldId [%d] channelId [%d].", e.Body.CharacterId, e.MapId, e.Instance, e.WorldId, e.ChannelId)
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(f, e.Body.CharacterId, despawnForSession(l)(ctx)(wp)(e.Body.CharacterId))
		if err != nil {
			l.WithError(err).Errorf("Unable to despawn character [%d] for characters in map [%d] instance [%s].", e.Body.CharacterId, e.MapId, e.Instance)
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
					return session.Announce(l)(ctx)(wp)(writer.ReactorSpawn)(writer.ReactorSpawnBody()(r))(s)
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

func handleStatusEventWeatherStart(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.WeatherStart]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.WeatherStart]) {
		if e.Type != _map3.EventTopicMapStatusTypeWeatherStart {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Weather started in map [%d] instance [%s] with item [%d].", e.MapId, e.Instance, e.Body.ItemId)
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(writer.FieldEffectWeather)(writer.FieldEffectWeatherStartBody(l)(e.Body.ItemId, e.Body.Message)))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast weather start to map [%d] instance [%s].", e.MapId, e.Instance)
		}
	}
}

func handleStatusEventWeatherEnd(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.WeatherEnd]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.WeatherEnd]) {
		if e.Type != _map3.EventTopicMapStatusTypeWeatherEnd {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Weather ended in map [%d] instance [%s].", e.MapId, e.Instance)
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(writer.FieldEffectWeather)(writer.FieldEffectWeatherEndBody(l)(e.Body.ItemId)))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast weather end to map [%d] instance [%s].", e.MapId, e.Instance)
		}
	}
}

package _map

import (
	"atlas-channel/chair"
	"atlas-channel/chalkboard"
	"atlas-channel/character"
	"atlas-channel/merchant"
	"atlas-channel/character/buff"
	cashData "atlas-channel/data/cash"
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
	"atlas-channel/saga"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/transport/route"
	"atlas-channel/weather"
	"context"
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	interactionpkt "github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	droppkt "github.com/Chronicle20/atlas/libs/atlas-packet/drop/clientbound"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	partycb "github.com/Chronicle20/atlas/libs/atlas-packet/party/clientbound"
	petpkt "github.com/Chronicle20/atlas/libs/atlas-packet/pet/clientbound"
	reactorpkt "github.com/Chronicle20/atlas/libs/atlas-packet/reactor/clientbound"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_status_event")(_map3.EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(_map3.EnvEventTopicMapStatus)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventWeatherStart(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventWeatherEnd(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapTimerStarted(sc, wp)))); err != nil {
					return err
				}
				return nil
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

// fetchOtherCharactersInMap returns a map of character models for all characters
// currently in the field, excluding the character with excludeId. Characters
// that cannot be fetched due to a 404 (stale registry entry) are skipped with
// a Warn log; only infrastructure errors are returned as hard failures.
func fetchOtherCharactersInMap(l logrus.FieldLogger, ctx context.Context, f field.Model, excludeId uint32) (map[uint32]character.Model, error) {
	ids, err := _map.NewProcessor(l, ctx).GetCharacterIdsInMap(f)
	if err != nil {
		return nil, err
	}

	cp := character.NewProcessor(l, ctx)
	cms := make(map[uint32]character.Model, len(ids))
	for _, id := range ids {
		if id == excludeId {
			continue
		}
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(id)
		if err != nil {
			if errors.Is(err, requests.ErrNotFound) {
				l.Warnf("Skipping stale registry entry: character [%d] not found in atlas-character.", id)
				continue
			}
			return nil, err
		}
		cms[id] = c
	}
	return cms, nil
}

// SpawnForSelf sends all per-character render packets to the entering session s:
// other characters visible in the field, their pets, NPCs, monsters, drops,
// reactors, chalkboards, chairs, merchants, boat state, clock, party HP, and
// active weather. It does NOT notify other players that s has arrived — that
// is enterMap's responsibility.
//
// Exported so that paths that write SetField (session bootstrap, warpCharacter)
// can call it synchronously after writing SetField to guarantee ordering.
func SpawnForSelf(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, f field.Model) error {
	return func(s session.Model, f field.Model) error {
		l.Debugf("SpawnForSelf: character [%d] in map [%d] instance [%s].", s.CharacterId(), f.MapId(), f.Instance())

		cms, err := fetchOtherCharactersInMap(l, ctx, f, s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("SpawnForSelf: unable to retrieve character details for characters in map.")
			return err
		}

		// spawn other characters for incoming (synchronous — must complete before goroutines so
		// client sees other players before world objects)
		for k, v := range cms {
			if k != s.CharacterId() {
				kg, _ := guild.NewProcessor(l, ctx).GetByMemberId(k)
				if err = spawnCharacterForSession(l)(ctx)(wp)(v, kg, false)(s); err != nil {
					l.WithError(err).Errorf("SpawnForSelf: unable to spawn character [%d] for [%d]", v.Id(), s.CharacterId())
				}
			}
		}

		go func() {
			for k, v := range cms {
				if k != s.CharacterId() {
					for _, p := range v.Pets() {
						if p.Slot() >= 0 {
							if err := session.Announce(l)(ctx)(wp)(petpkt.PetActivatedWriter)(petpkt.PetSpawnBody(p.OwnerId(), p.Slot(), p.TemplateId(), p.Name(), uint64(p.Id()), p.X(), p.Y(), p.Stance(), uint16(p.Fh())))(s); err != nil {
								l.WithError(err).Errorf("SpawnForSelf: unable to spawn character [%d] pet for [%d]", k, s.CharacterId())
							}
						}
					}
				}
			}
		}()

		go func() {
			if err := npc2.NewProcessor(l, ctx).ForEachInMap(f.MapId(), spawnNPCForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Errorf("SpawnForSelf: unable to spawn npcs for character [%d].", s.CharacterId())
			}
		}()

		go func() {
			if err := monster.NewProcessor(l, ctx).ForEachInMap(f, spawnMonsterForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn monsters for character [%d].", s.CharacterId())
			}
		}()

		go func() {
			if err := drop.NewProcessor(l, ctx).ForEachInMap(f, spawnDropsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn drops for character [%d].", s.CharacterId())
			}
		}()

		go func() {
			if err := reactor.NewProcessor(l, ctx).ForEachInMap(f, spawnReactorsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn reactors for character [%d].", s.CharacterId())
			}
		}()

		go func() {
			if err := chalkboard.NewProcessor(l, ctx).ForEachInMap(f, spawnChalkboardsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn chalkboards for character [%d].", s.CharacterId())
			}
		}()

		go func() {
			if err := chair.NewProcessor(l, ctx).ForEachInMap(f, spawnChairsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn chairs for character [%d].", s.CharacterId())
			}
		}()

		go func() {
			if err := merchant.NewProcessor(l, ctx).ForEachInField(f, spawnMerchantsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn merchants for character [%d].", s.CharacterId())
			}
		}()

		go func() {
			cp := character.NewProcessor(l, ctx)
			cd, err := cp.GetById(cp.PartyDecorator)(s.CharacterId())
			if err != nil || !cd.InParty() {
				return
			}
			pmp := model.FixedProvider(cd.Party())
			imf := party.OtherMemberInMap(s.Field(), s.CharacterId())
			oip := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(pmp))
			_ = session.NewProcessor(l, ctx).ForEachByCharacterId(s.Field().Channel())(oip, session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(s.CharacterId(), cd.Hp(), cd.MaxHp()).Encode))
			_ = model.ForEachSlice(oip, func(oid uint32) error {
				oc, oerr := cp.GetById()(oid)
				if oerr != nil {
					if errors.Is(oerr, requests.ErrNotFound) {
						l.Warnf("SpawnForSelf: skipping party HP for stale character [%d].", oid)
						return nil
					}
					return oerr
				}
				return session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(oid, oc.Hp(), oc.MaxHp()).Encode)(s)
			}, model.ParallelExecute())
		}()

		go func() {
			md, err := mapData.NewProcessor(l, ctx).GetById(f.MapId())
			if err != nil {
				l.WithError(err).Errorf("SpawnForSelf: unable to retrieve map data for map [%d].", f.MapId())
				return
			}
			if md.Clock() {
				now := time.Now()
				_ = session.Announce(l)(ctx)(wp)(fieldcb.ClockWriter)(fieldcb.NewTownClock(byte(now.Hour()), byte(now.Minute()), byte(now.Second())).Encode)(s)
			}
		}()

		go func() {
			hasShip, err := route.NewProcessor(l, ctx).IsBoatInMap(f.MapId())
			if err != nil {
				l.WithError(err).Errorf("SpawnForSelf: unable to retrieve boat data for map [%d].", f.MapId())
				return
			}
			if hasShip {
				_ = session.Announce(l)(ctx)(wp)(fieldcb.FieldTransportStateWriter)(fieldcb.NewFieldTransport(fieldcb.TransportStateEnter1, false).Encode)(s)
			} else {
				_ = session.Announce(l)(ctx)(wp)(fieldcb.FieldTransportStateWriter)(fieldcb.NewFieldTransport(fieldcb.TransportStateMove1, false).Encode)(s)
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
			_ = session.Announce(l)(ctx)(wp)(fieldcb.ClockWriter)(fieldcb.NewTimerClock(uint32(timer.Duration().Seconds())).Encode)(s)
		}()

		go func() {
			we, werr := weather.NewProcessor(l, ctx).GetActive(f)
			if werr != nil {
				return
			}
			_ = session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWeatherWriter)(fieldcb.NewFieldEffectWeatherStart(we.ItemId, we.Message).Encode)(s)

			ci, cerr := cashData.NewProcessor(l, ctx).GetById(we.ItemId)
			if cerr != nil {
				return
			}
			if ci.BgmPath != "" {
				_ = session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectBackgroundMusicBody(ci.BgmPath))(s)
			}
			if ci.StateChangeItem > 0 {
				applyConsumableEffectSaga(l, saga.NewProcessor(l, ctx), s.CharacterId(), f, ci.StateChangeItem)
			}
		}()

		return nil
	}
}

// enterMap notifies other players in the field that a new character has arrived,
// and spawns the entering character's pets for those players. The "spawn world
// for self" work (NPCs, monsters, other characters, etc.) is handled by
// SpawnForSelf, which must be called separately by the SetField-writing path.
func enterMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model) model.Operator[session.Model] {
	return func(f field.Model) model.Operator[session.Model] {
		return func(s session.Model) error {
			l.Debugf("Processing character [%d] entering map [%d] instance [%s].", s.CharacterId(), f.MapId(), f.Instance())

			// fetch the entering character's own model for spawning to others
			cp := character.NewProcessor(l, ctx)
			self, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("enterMap: unable to fetch self character [%d].", s.CharacterId())
				return err
			}
			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())

			// collect other character IDs already in the map
			ids, err := _map.NewProcessor(l, ctx).GetCharacterIdsInMap(f)
			if err != nil {
				l.WithError(err).Errorf("enterMap: failed to fetch characters in map [%d] instance [%s] for world [%d] and channel [%d]: aborting inter-character notifications.", f.MapId(), f.Instance(), f.WorldId(), f.ChannelId())
				return err
			}

			// spawn new character for others — skip entries whose session is gone
			for _, k := range ids {
				if k == s.CharacterId() {
					continue
				}
				if err = session.NewProcessor(l, ctx).IfPresentByCharacterId(s.Field().Channel())(k, spawnCharacterForSession(l)(ctx)(wp)(self, g, true)); err != nil {
					if errors.Is(err, requests.ErrNotFound) {
						l.Warnf("enterMap: skipping stale session entry for character [%d].", k)
						continue
					}
					l.WithError(err).Errorf("enterMap: unable to spawn character [%d] for [%d] — continuing.", k, s.CharacterId())
				}
			}

			// spawn self's pets for every other player in the map
			go func() {
				for _, k := range ids {
					if k == s.CharacterId() {
						continue
					}
					for _, p := range self.Pets() {
						if p.Slot() >= 0 {
							if err := session.NewProcessor(l, ctx).IfPresentByCharacterId(s.Field().Channel())(k, session.Announce(l)(ctx)(wp)(petpkt.PetActivatedWriter)(petpkt.PetSpawnBody(p.OwnerId(), p.Slot(), p.TemplateId(), p.Name(), uint64(p.Id()), p.X(), p.Y(), p.Stance(), uint16(p.Fh())))); err != nil {
								l.WithError(err).Errorf("enterMap: unable to spawn character [%d] pet for [%d]", s.CharacterId(), k)
							}
							excludeIds := make([]uint32, len(p.Excludes()))
							for i, e := range p.Excludes() {
								excludeIds[i] = e.ItemId()
							}
							if err := session.Announce(l)(ctx)(wp)(petpkt.PetExcludeResponseWriter)(petpkt.NewPetExcludeResponse(p.OwnerId(), p.Slot(), uint64(p.Id()), excludeIds).Encode)(s); err != nil {
								l.WithError(err).Errorf("enterMap: unable to announce pet [%d] exclusion list to character [%d].", p.Id(), s.CharacterId())
							}
						}
					}
				}
			}()

			// "spawn world for self" (SpawnForSelf) is handled by the SetField-writing
			// path (session bootstrap and warpCharacter) to guarantee packet ordering.
			// enterMap only handles the inter-character notifications.
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

					return session.Announce(l)(ctx)(wp)(charpkt.CharacterSpawnWriter)(writer.CharacterSpawnBody(c, bs, g, enteringField))(s)
				}
			}
		}
	}
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
				return session.Announce(l)(ctx)(wp)(charpkt.CharacterDespawnWriter)(charpkt.NewCharacterDespawn(id).Encode)
			}
		}
	}
}

func spawnNPCForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc2.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc2.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[npc2.Model] {
			return func(s session.Model) model.Operator[npc2.Model] {
				return func(n npc2.Model) error {
					err := session.Announce(l)(ctx)(wp)(npcpkt.NpcSpawnWriter)(npcpkt.NewNpcSpawn(n.Id(), n.Template(), n.X(), n.CY(), int32(n.F()), n.Fh(), n.RX0(), n.RX1()).Encode)(s)
					if err != nil {
						return err
					}
					return session.Announce(l)(ctx)(wp)(npcpkt.NpcSpawnRequestControllerWriter)(npcpkt.NewNpcSpawnRequestController(n.Id(), n.Template(), n.X(), n.CY(), int32(n.F()), n.Fh(), n.RX0(), n.RX1(), true).Encode)(s)
				}
			}
		}
	}
}

// spawnMonsterForSession sends the per-mob spawn packet to the entering session
// and, when the entering character is the current controller, also re-issues the
// MonsterControl packet that grants client-side ownership.
//
// Why the re-issue: on cash-shop return atlas-monsters reassigns control via
// CharacterEnter (MAP_STATUS) and emits StartControl events that atlas-channel
// turns into MonsterControl packets — but those land on the wire ~1s before the
// matching mob spawn packets that this function emits. The v83 client drops
// (or ignores) MonsterControl for an unknown uniqueId, so by the time the spawn
// renders the mob it has no formal owner. Sending MonsterControl right after
// the spawn closes that gap deterministically; the earlier Kafka-driven
// MonsterControl is at worst a harmless duplicate.
func spawnMonsterForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[monster.Model] {
			return func(s session.Model) model.Operator[monster.Model] {
				return func(m monster.Model) error {
					if err := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterSpawnWriter)(writer.SpawnMonsterBody(m, false))(s); err != nil {
						return err
					}
					if m.ControlCharacterId() == s.CharacterId() {
						if err := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterControlWriter)(writer.StartControlMonsterBody(m, m.ControllerHasAggro()))(s); err != nil {
							l.WithError(err).Errorf("SpawnForSelf: unable to re-issue MonsterControl for character [%d] mob [%d].", s.CharacterId(), m.UniqueId())
						}
					}
					return nil
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
					return session.Announce(l)(ctx)(wp)(droppkt.DropSpawnWriter)(droppkt.NewDropSpawn(
						droppkt.DropEnterTypeExisting, d.Id(), d.Meso(), d.ItemId(),
						d.Owner(), d.Type(), d.X(), d.Y(), d.DropperId(),
						d.DropperX(), d.DropperY(), 0, d.CharacterDrop(),
					).Encode)(s)
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
					return session.Announce(l)(ctx)(wp)(reactorpkt.ReactorSpawnWriter)(reactorpkt.NewReactorSpawn(r.Id(), r.Classification(), r.State(), r.X(), r.Y(), r.Direction(), r.Name()).Encode)(s)
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
					return session.Announce(l)(ctx)(wp)(charpkt.ChalkboardUseWriter)(charpkt.NewChalkboardUse(c.Id(), c.Message()).Encode)(s)
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
					return session.Announce(l)(ctx)(wp)(charpkt.CharacterShowChairWriter)(charpkt.NewCharacterChairShow(c.CharacterId(), c.Id()).Encode)(s)
				}
			}
		}
	}
}

func spawnMerchantsForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[merchant.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[merchant.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[merchant.Model] {
			return func(s session.Model) model.Operator[merchant.Model] {
				return func(m merchant.Model) error {
					miniRoomType := interactionpkt.MerchantShopMiniRoomType
					if m.ShopType() == 1 {
						miniRoomType = interactionpkt.PersonalShopMiniRoomType
					}
					mr := &interactionpkt.MiniRoomBase{
						MiniRoomTypeVal: miniRoomType,
						Title:           m.Title(),
						CapacityVal:     4,
						OwnerId:         m.CharacterId(),
						VisitorCount:    byte(len(m.Visitors())),
						VisitorList:     []interactionpkt.MiniRoomVisitor{},
					}
					return session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(mr.Spawn(m.CharacterId()))(s)
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
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWeatherWriter)(fieldcb.NewFieldEffectWeatherStart(e.Body.ItemId, e.Body.Message).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast weather start to map [%d] instance [%s].", e.MapId, e.Instance)
		}

		go applyWeatherEffects(l, ctx, wp, f, e.Body.ItemId)
	}
}

func applyWeatherEffects(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, itemId uint32) {
	ci, err := cashData.NewProcessor(l, ctx).GetById(itemId)
	if err != nil {
		l.WithError(err).Debugf("Unable to retrieve cash item [%d] data for weather effects.", itemId)
		return
	}

	if ci.BgmPath != "" {
		_ = _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectBackgroundMusicBody(ci.BgmPath)))
	}

	if ci.StateChangeItem == 0 {
		return
	}

	ids, err := _map.NewProcessor(l, ctx).GetCharacterIdsInMap(f)
	if err != nil {
		l.WithError(err).Errorf("Unable to get character IDs in map [%d] instance [%s] for weather buff.", f.MapId(), f.Instance())
		return
	}

	sp := saga.NewProcessor(l, ctx)
	for _, id := range ids {
		applyConsumableEffectSaga(l, sp, id, f, ci.StateChangeItem)
	}
}

func applyConsumableEffectSaga(l logrus.FieldLogger, sp saga.Processor, characterId uint32, f field.Model, itemId uint32) {
	now := time.Now()
	s := saga.Saga{
		TransactionId: uuid.New(),
		SagaType:      saga.FieldEffectUse,
		InitiatedBy:   "WEATHER",
		Steps: []saga.Step{
			{
				StepId:    "apply_consumable_effect",
				Status:    saga.Pending,
				Action:    saga.ApplyConsumableEffect,
				Payload:   saga.ApplyConsumableEffectPayload{CharacterId: characterId, WorldId: f.WorldId(), ChannelId: f.ChannelId(), ItemId: itemId},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	err := sp.Create(s)
	if err != nil {
		l.WithError(err).Errorf("Unable to create apply consumable effect saga for character [%d] item [%d].", characterId, itemId)
	}
}

func handleStatusEventMapTimerStarted(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.MapTimerStarted]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.MapTimerStarted]) {
		if e.Type != _map3.EventTopicMapStatusTypeMapTimerStarted {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("MAP_TIMER_STARTED for character [%d] map [%d] seconds [%d].", e.Body.CharacterId, e.MapId, e.Body.Seconds)
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, session.Announce(l)(ctx)(wp)(fieldcb.ClockWriter)(fieldcb.NewTimerClock(e.Body.Seconds).Encode))
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
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWeatherWriter)(fieldcb.NewFieldEffectWeatherEnd(e.Body.ItemId).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast weather end to map [%d] instance [%s].", e.MapId, e.Instance)
		}
	}
}

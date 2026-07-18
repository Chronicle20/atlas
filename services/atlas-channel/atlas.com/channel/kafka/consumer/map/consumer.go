package _map

import (
	"atlas-channel/chair"
	"atlas-channel/chalkboard"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	cashData "atlas-channel/data/cash"
	mapData "atlas-channel/data/map"
	npc2 "atlas-channel/data/npc"
	"atlas-channel/door"
	"atlas-channel/drop"
	"atlas-channel/guild"
	consumer2 "atlas-channel/kafka/consumer"
	_map3 "atlas-channel/kafka/message/map"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/merchant"
	"atlas-channel/minigame"
	"atlas-channel/monster"
	controllernpc "atlas-channel/npc/controller"
	"atlas-channel/party"
	"atlas-channel/party/hpsync"
	"atlas-channel/party_quest"
	"atlas-channel/reactor"
	"atlas-channel/saga"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	summoncmd "atlas-channel/summon"
	"atlas-channel/transport/route"
	"atlas-channel/weather"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	doorcb "github.com/Chronicle20/atlas/libs/atlas-packet/door/clientbound"
	droppkt "github.com/Chronicle20/atlas/libs/atlas-packet/drop/clientbound"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	interactionpkt "github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	petpkt "github.com/Chronicle20/atlas/libs/atlas-packet/pet/clientbound"
	reactorpkt "github.com/Chronicle20/atlas/libs/atlas-packet/reactor/clientbound"
	summonpkt "github.com/Chronicle20/atlas/libs/atlas-packet/summon/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("map_status_event")(_map3.EnvEventTopicMapStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(_map3.EnvEventTopicMapStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterEnter(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCharacterExit(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventWeatherStart(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventWeatherEnd(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapTimerStarted(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
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

		routine.Go(l, ctx, func(_ context.Context) {
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
		})

		// spawn the entering character's OWN spawned pets back to themselves.
		// enterMap spawns self's pets to other players, and the loop above spawns
		// other players' pets to self, but on a fresh field entry (login, map
		// change, cash-shop return) nothing re-sends the owner's own pet to the
		// owner. Without this the pet stays invisible to its owner even though it
		// is still spawned (slot >= 0).
		routine.Go(l, ctx, func(_ context.Context) {
			cp := character.NewProcessor(l, ctx)
			self, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("SpawnForSelf: unable to fetch self [%d] for own-pet spawn.", s.CharacterId())
				return
			}
			for _, p := range self.Pets() {
				if p.Slot() >= 0 {
					if err := session.Announce(l)(ctx)(wp)(petpkt.PetActivatedWriter)(petpkt.PetSpawnBody(p.OwnerId(), p.Slot(), p.TemplateId(), p.Name(), uint64(p.Id()), p.X(), p.Y(), p.Stance(), uint16(p.Fh())))(s); err != nil {
						l.WithError(err).Errorf("SpawnForSelf: unable to spawn own pet for character [%d].", s.CharacterId())
					}
				}
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := npc2.NewProcessor(l, ctx).ForEachInMap(f.MapId(), spawnNPCForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Errorf("SpawnForSelf: unable to spawn npcs for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := monster.NewProcessor(l, ctx).ForEachInMap(f, spawnMonsterForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn monsters for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := summoncmd.NewProcessor(l, ctx).ForEachInMap(f, spawnSummonForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn summons for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := drop.NewProcessor(l, ctx).ForEachInMap(f, spawnDropsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn drops for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := reactor.NewProcessor(l, ctx).ForEachInMap(f, spawnReactorsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn reactors for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := door.NewProcessor(l, ctx).ForEachInMap(f, spawnDoorsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn doors for character [%d].", s.CharacterId())
			}
		})
		routine.Go(l, ctx, func(_ context.Context) {
			// Town side: render the walkable town door to a player entering the
			// return town (FR-3.2/FR-3.4) — the area-side spawn above misses it.
			spawnTownDoorsForSession(l, ctx, wp, s)
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := chalkboard.NewProcessor(l, ctx).ForEachInMap(f, spawnChalkboardsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn chalkboards for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := chair.NewProcessor(l, ctx).ForEachInMap(f, spawnChairsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn chairs for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := merchant.NewProcessor(l, ctx).ForEachInField(f, spawnMerchantsForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn merchants for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := minigame.NewProcessor(l, ctx).ForEachInField(f, spawnMiniGamesForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn mini-games for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			if err := hpsync.Sync(l, ctx, wp, s.Field(), s.CharacterId()); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to sync party member HP for character [%d].", s.CharacterId())
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
			md, err := mapData.NewProcessor(l, ctx).GetById(f.MapId())
			if err != nil {
				l.WithError(err).Errorf("SpawnForSelf: unable to retrieve map data for map [%d].", f.MapId())
				return
			}
			if md.Clock() {
				now := time.Now()
				_ = session.Announce(l)(ctx)(wp)(fieldcb.ClockWriter)(fieldcb.NewTownClock(byte(now.Hour()), byte(now.Minute()), byte(now.Second())).Encode)(s)
			}
		})

		routine.Go(l, ctx, func(_ context.Context) {
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
		})

		routine.Go(l, ctx, func(_ context.Context) {
			timer, terr := party_quest.NewProcessor(l, ctx).GetTimerByCharacterId(s.CharacterId())
			if terr != nil {
				return
			}
			if timer.Duration() <= 0 {
				return
			}
			_ = session.Announce(l)(ctx)(wp)(fieldcb.ClockWriter)(fieldcb.NewTimerClock(uint32(timer.Duration().Seconds())).Encode)(s)
		})

		routine.Go(l, ctx, func(_ context.Context) {
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
		})

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
			routine.Go(l, ctx, func(_ context.Context) {
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
			})

			// "spawn world for self" (SpawnForSelf) is handled by the SetField-writing
			// path (session bootstrap and warpCharacter) to guarantee packet ordering.
			// enterMap only handles the inter-character notifications.
			return nil
		}
	}
}

// spawnSummonForSession sends a SummonSpawn for an existing summon to the entering
// session s. animated=false (the summon is already present — no spawn animation);
// stance=0 matches a fresh cast (which also spawns at stance 0). The summon's
// movement corrects on the next broadcast SummonMove.
func spawnSummonForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[summoncmd.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[summoncmd.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[summoncmd.Model] {
			return func(s session.Model) model.Operator[summoncmd.Model] {
				return func(m summoncmd.Model) error {
					return session.Announce(l)(ctx)(wp)(summonpkt.SummonSpawnWriter)(
						writer.SummonSpawnBody(m.OwnerCharacterId(), m.Id(), m.SkillId(), m.SkillLevel(), m.X(), m.Y(), 0, m.MovementType(), m.IsPuppet(), false))(s)
				}
			}
		}
	}
}

// emitCharacterSpawn sends the CharacterSpawn packet for c to session s using
// the already-fetched buff state bs. This is the single wire-emit choke point
// shared by the gated (spawnCharacterForSession) and ungated
// (spawnCharacterForSessionRevealed) operators below — it contains NO
// suppression logic itself; callers decide whether/when a spawn is gated.
func emitCharacterSpawn(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(c character.Model, bs []buff.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(c character.Model, bs []buff.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(c character.Model, bs []buff.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
			return func(c character.Model, bs []buff.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
				return func(s session.Model) error {
					return session.Announce(l)(ctx)(wp)(charpkt.CharacterSpawnWriter)(writer.CharacterSpawnBody(c, bs, g, enteringField))(s)
				}
			}
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

					// GM-hide suppression (task-156). A character hidden via the
					// SuperGM Hide skill must not be spawned to any OTHER viewer.
					// This is the single choke point for every character spawn —
					// enterMap->others and SpawnForSelf-of-others both pass here —
					// so a viewer entering while a GM is hidden never sees the
					// spawn (race-safe: the check is in the same path that emits
					// it). c is never the viewer's own character (both callers
					// skip k == s.CharacterId()), so self-view is never suppressed.
					if buff.IsGmHidden(bs) {
						return nil
					}

					return emitCharacterSpawn(l)(ctx)(wp)(c, bs, g, enteringField)(s)
				}
			}
		}
	}
}

// spawnCharacterForSessionRevealed sends a CharacterSpawn for c to session s
// with NO GM-hide suppression check. It exists solely for the "hide off"
// reveal path (SpawnCharacterInMap below).
//
// It is intentionally ungated: the reveal caller (skill/handler/hide) has
// just PRODUCED an async CANCEL command for the hide buff to atlas-buffs —
// that cancellation is Kafka-mediated and eventually-consistent, not a
// synchronous local mutation. A gated read here (buff.NewProcessor(...).
// GetByCharacterId + buff.IsGmHidden, as spawnCharacterForSession above does)
// would very likely still observe the not-yet-cancelled hide buff and wrongly
// re-suppress the very spawn that is supposed to un-hide the character,
// leaving the GM permanently invisible to anyone already in the map. The
// reveal caller is the one deciding to end the hide, so this path must force
// the spawn rather than depend on server-side buff state catching up.
//
// Including the still-present DARK_SIGHT buff in bs is safe: DARK_SIGHT
// foreign-encodes as a no-op on a remote character, and nothing in the spawn
// packet itself hides a remote character — only the server declining to emit
// CharacterSpawn does. No filtering of bs is needed.
//
// NO reference to buff.IsGmHidden appears in this function — that absence is
// the structural guarantee that the reveal path can never be gated.
func spawnCharacterForSessionRevealed(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
			return func(c character.Model, g guild.Model, enteringField bool) model.Operator[session.Model] {
				return func(s session.Model) error {
					bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(c.Id())
					if err != nil {
						bs = make([]buff.Model, 0)
					}

					return emitCharacterSpawn(l)(ctx)(wp)(c, bs, g, enteringField)(s)
				}
			}
		}
	}
}

// DespawnCharacterInMap broadcasts a CharacterDespawn for characterId to every
// OTHER session in field f — the "hide on" half of the GM-hide toggle. Reuses
// the existing per-session despawn operator so the packet matches a normal exit.
func DespawnCharacterInMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model, characterId uint32) error {
	return func(f field.Model, characterId uint32) error {
		return _map.NewProcessor(l, ctx).ForOtherSessionsInMap(f, characterId, despawnForSession(l)(ctx)(wp)(characterId))
	}
}

// SpawnCharacterInMap broadcasts a CharacterSpawn for characterId to every OTHER
// session in field f — the "hide off" (reveal) half of the GM-hide toggle. It
// uses spawnCharacterForSessionRevealed (NOT the gated spawnCharacterForSession)
// so the spawn packet is byte-identical to a normal map-entry spawn (buffs +
// guild + enteringField=false, since the caster is already standing in the
// map) while being immune to the async-cancel race: the hide-buff CANCEL this
// caller just produced to atlas-buffs is eventually-consistent, so a gated
// read here could still observe the stale hide buff and wrongly re-suppress
// the reveal, leaving the GM permanently invisible. See
// spawnCharacterForSessionRevealed's doc comment for the full rationale.
func SpawnCharacterInMap(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model, characterId uint32) error {
	return func(f field.Model, characterId uint32) error {
		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(characterId)
		if err != nil {
			return err
		}
		g, _ := guild.NewProcessor(l, ctx).GetByMemberId(characterId)
		return _map.NewProcessor(l, ctx).ForOtherSessionsInMap(f, characterId, spawnCharacterForSessionRevealed(l)(ctx)(wp)(c, g, false))
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
				cp := controllernpc.NewProcessor(l, ctx)
				return func(n npc2.Model) error {
					err := session.Announce(l)(ctx)(wp)(npcpkt.NpcSpawnWriter)(npcpkt.NewNpcSpawn(n.Id(), n.Template(), n.X(), n.CY(), int32(n.F()), n.Fh(), n.RX0(), n.RX1()).Encode)(s)
					if err != nil {
						return err
					}
					// Single-controller election (task-176, FR-5.2/FR-5.4):
					// claim synchronously so NpcSpawn -> grant land on the
					// same session in order; non-controllers get spawn only.
					claimed, cerr := cp.TryClaim(s.Field(), n.Id(), s.CharacterId())
					if cerr != nil {
						l.WithError(cerr).Warnf("NPC-controller claim failed for NPC [%d]; session [%d] gets spawn only.", n.Id(), s.CharacterId())
						return nil
					}
					if !claimed {
						return nil
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

// doorAnnounce is the session.Announce seam for door packets, extracted as a
// package-level var so tests can stub it without a real socket writer. The
// writerName parameter identifies the writer (e.g. SpawnDoorWriter) for test
// assertions; the real implementation calls session.Announce with the given
// writerName and body.
var doorAnnounce = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, writerName string, enc packet.Encode, s session.Model) error {
	return session.Announce(l)(ctx)(wp)(writerName)(enc)(s)
}

// spawnDoorsForSession returns a door.Model operator that announces the
// area-side door to the arriving session (FR-3.4). The area door is a plain
// ranged map object — shown to EVERY session in the map, like a monster (no
// party filter). Party membership only gates door ENTRY and the town-portal
// array, not area visibility. For AREA-side doors (returned by
// door.Processor.ForEachInMap keyed on the area field), the wire packet is
// SpawnDoor(ownerCharacterId, areaX, areaY, launched=true); launched=true marks
// a late-join re-spawn vs. a first deploy. The wire "oid" is the owner character
// id.
//
// Town-side spawn (the walkable town door, clientbound spawnPortal, for a
// session entering the return town) is handled separately by
// spawnTownDoorsForSession — GetInField is keyed on the area field, so the
// town side is resolved by owner instead.
func spawnDoorsForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[door.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[door.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[door.Model] {
			return func(s session.Model) model.Operator[door.Model] {
				return func(d door.Model) error {
					return doorAnnounce(l, ctx, wp, doorcb.SpawnDoorWriter,
						writer.SpawnDoorBody(d.OwnerCharacterId(), d.AreaX(), d.AreaY(), true), s)
				}
			}
		}
	}
}

// townDoorsByOwnerFunc lists a character's live doors via the by-owner route.
// Package var so tests can stub it.
var townDoorsByOwnerFunc = func(l logrus.FieldLogger, ctx context.Context, ownerId uint32) ([]door.Model, error) {
	return door.NewProcessor(l, ctx).GetByOwner(ownerId)
}

// townSpawnPartyMembers returns the session's character plus its same-party
// member ids — the owners whose doors the session is eligible to see. Package
// var so tests can stub party membership.
var townSpawnPartyMembers = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) []uint32 {
	ids := []uint32{characterId}
	pm, err := party.NewProcessor(l, ctx).GetByMemberId(characterId)
	if err != nil {
		return ids
	}
	for _, m := range pm.Members() {
		if m.Id() != characterId {
			ids = append(ids, m.Id())
		}
	}
	return ids
}

// spawnTownDoorsForSession announces the TOWN-side door (clientbound spawnPortal)
// to a session entering its field, for every door owned by the session's party
// whose town side IS the entered map (FR-3.2 / FR-3.4). The area-side map-enter
// spawn (spawnDoorsForSession) only covers doors whose AREA field is the entered
// map; a player warping INTO the return town also needs the walkable town door
// rendered. v83 draws the town door from spawnPortal positioned at the town door
// portal (0x80+slot) resolved at cast. Resolved by owner (no by-town REST route),
// de-duplicated across party members.
func spawnTownDoorsForSession(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) {
	f := s.Field()
	seen := map[uint32]struct{}{}
	for _, owner := range townSpawnPartyMembers(l, ctx, s.CharacterId()) {
		doors, err := townDoorsByOwnerFunc(l, ctx, owner)
		if err != nil {
			continue
		}
		for _, d := range doors {
			if d.TownMapId() != f.MapId() {
				continue
			}
			if d.WorldId() != f.WorldId() || d.ChannelId() != f.ChannelId() {
				continue
			}
			if _, dup := seen[d.AreaDoorId()]; dup {
				continue
			}
			seen[d.AreaDoorId()] = struct{}{}
			_ = doorAnnounce(l, ctx, wp, doorcb.SpawnPortalWriter,
				writer.SpawnPortalBody(d.TownMapId(), d.MapId(), d.AreaX(), d.AreaY()), s)
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
					if m.ShopType() == merchant.HiredMerchantShopType {
						// Hired merchant renders as a standalone employee NPC (D1); spawn
						// it to the entering player.
						ownerName := ""
						if c, err := character.NewProcessor(l, ctx).GetById()(m.CharacterId()); err != nil {
							l.WithError(err).Warnf("Unable to resolve hired-merchant owner [%d] name for field spawn.", m.CharacterId())
						} else {
							ownerName = c.Name()
						}
						spawn := merchant.ToEmployeeSpawn(m, ownerName)
						return session.Announce(l)(ctx)(wp)(merchantcb.MerchantEmployeeSpawnWriter)(spawn.Encode)(s)
					}
					// Personal store: box on the owner's avatar.
					mr := &interactionpkt.MiniRoomBase{
						MiniRoomTypeVal: interactionpkt.PersonalShopMiniRoomType,
						// Id = dwMiniRoomSN: the client echoes it as the visit
						// serialNumber and the server resolves via
						// GetByCharacterId(serialNumber), so it must be the owner's
						// character id (task-127; see merchant consumer note).
						Id:           m.CharacterId(),
						Title:        m.Title(),
						Spec:         merchant.StoreSkinSpec(m.PermitItemId()),
						CapacityVal:  4,
						OwnerId:      m.CharacterId(),
						VisitorCount: byte(len(m.Visitors())),
						VisitorList:  []interactionpkt.MiniRoomVisitor{},
					}
					return session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(mr.Spawn(m.CharacterId()))(s)
				}
			}
		}
	}
}

// spawnMiniGamesForSession announces the UPDATE_CHAR_BOX balloon for every
// mini-game room (Omok/Match Cards) currently registered in the field to the
// entering session, mirroring the merchant/shop balloon spawn above. Capacity
// is fixed at 2 for both game dialogs (design §5; matches gameRoomCapacity in
// kafka/consumer/minigame/consumer.go).
func spawnMiniGamesForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[minigame.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[minigame.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[minigame.Model] {
			return func(s session.Model) model.Operator[minigame.Model] {
				return func(m minigame.Model) error {
					return session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(interactioncb.MiniRoomBalloonBody(m.OwnerId(), m.RoomType(), m.Id(), m.Title(), m.HasPassword(), m.PieceType(), m.Occupancy(), 2, m.InProgress()))(s)
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

		routine.Go(l, ctx, func(_ context.Context) {
			applyWeatherEffects(l, ctx, wp, f, e.Body.ItemId)
		})
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

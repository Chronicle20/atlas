package session

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/key"
	"atlas-channel/guild"
	consumer2 "atlas-channel/kafka/consumer"
	mapconsumer "atlas-channel/kafka/consumer/map"
	session2 "atlas-channel/kafka/message/account/session"
	"atlas-channel/listener"
	"atlas-channel/macro"
	"atlas-channel/maps/location"
	"atlas-channel/note"
	"atlas-channel/ring"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"atlas-channel/world"
	"context"
	"errors"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	buddypkt "github.com/Chronicle20/atlas/libs/atlas-packet/buddy"
	buddyCB "github.com/Chronicle20/atlas/libs/atlas-packet/buddy/clientbound"
	channelpkt "github.com/Chronicle20/atlas/libs/atlas-packet/channel/clientbound"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	guildpkt "github.com/Chronicle20/atlas/libs/atlas-packet/guild"
	guildcb "github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound"
	notepkt "github.com/Chronicle20/atlas/libs/atlas-packet/note"
	notecb "github.com/Chronicle20/atlas/libs/atlas-packet/note/clientbound"
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("account_session_status_event")(session2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var err error
				var handles []listener.HandlerHandle
				t, err = topic.EnvProvider(l)(session2.EnvEventStatusTopic)()
				if err != nil {
					return nil, err
				}
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleError(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleChannelChange(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handlePlayerLoggedIn(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleError(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[session2.ErrorStatusEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[session2.ErrorStatusEventBody]) {
		if e.Type != session2.EventStatusTypeError {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByIdInWorld(e.SessionId, sc.Channel(), announceError(l)(ctx)(wp)(e.Body.Code))
	}
}

func announceError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(reason string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(reason string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(reason string) model.Operator[session.Model] {
			return func(reason string) model.Operator[session.Model] {
				return func(s session.Model) error {
					l.Errorf("Unable to update session for character [%d] attempting to switch to channel.", s.CharacterId())
					return session.NewProcessor(l, ctx).Destroy(s)
				}
			}
		}
	}
}

func handleChannelChange(sc server.Model, wp writer.Producer) message.Handler[session2.StatusEvent[session2.StateChangedEventBody[model2.ChannelChange]]] {
	return func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[session2.StateChangedEventBody[model2.ChannelChange]]) {
		if e.Type != session2.EventStatusTypeStateChanged {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if len(e.Body.Params.IPAddress) <= 0 {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByIdInWorld(e.SessionId, sc.Channel(), processChannelChangeReturn(l)(ctx)(wp)(e.AccountId, e.Body.State, e.Body.Params))
	}
}

func processChannelChangeReturn(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(accountId uint32, state uint8, params model2.ChannelChange) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(accountId uint32, state uint8, params model2.ChannelChange) model.Operator[session.Model] {
		return func(wp writer.Producer) func(accountId uint32, state uint8, params model2.ChannelChange) model.Operator[session.Model] {
			return func(accountId uint32, state uint8, params model2.ChannelChange) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(channelpkt.ChannelChangeWriter)(channelpkt.NewChannelChange(params.IPAddress, params.Port).Encode)
			}
		}
	}
}

func handlePlayerLoggedIn(sc server.Model, wp writer.Producer) message.Handler[session2.StatusEvent[session2.StateChangedEventBody[model2.SetField]]] {
	return func(l logrus.FieldLogger, ctx context.Context, e session2.StatusEvent[session2.StateChangedEventBody[model2.SetField]]) {
		if e.Type != session2.EventStatusTypeStateChanged {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByIdInWorld(e.SessionId, sc.Channel(), processStateReturn(l)(ctx)(wp)(e.AccountId, e.Body.State, e.Body.Params))
	}
}

// claimEnableAnnouncer sends the two claim-enable bootstrap packets - the
// client keeps its CUIClaim window disabled until m_bClaimSvrConnected is
// set (ClaimSvrStatusChanged) and an availability window arrives
// (ClaimAvailableTime; 0,0 = always open, a client-side special case, not an
// operation code - see writer.ClaimAvailableTimeBody).
//
// Config presence IS the feature gate, mirroring the reportAnnouncer seam in
// kafka/consumer/report/consumer.go: a v61 tenant supports sue but not
// claim, and jms/gms-92 tenants support neither, so those tenants have no
// ClaimSvrStatusChanged/ClaimAvailableTime writer in their socket-config
// template. Pre-checking the status writer's lookup before sending anything
// means those tenants skip both sends (debug log), never surfacing an error
// on every single login. A tenant that maps the status writer but not the
// availability writer (an inconsistent template, not the expected on/off
// pairing) still logs the second failure at error level so it isn't lost
// silently.
func claimEnableAnnouncer(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, characterId uint32) {
	if _, err := wp(reportcb.ClaimSvrStatusChangedWriter); err != nil {
		l.Debugf("Tenant configuration has no writer [%s] mapped; claim UI stays disabled for character [%d].", reportcb.ClaimSvrStatusChangedWriter, characterId)
		return
	}
	err := session.Announce(l)(ctx)(wp)(reportcb.ClaimSvrStatusChangedWriter)(writer.ClaimSvrStatusChangedBody(true))(s)
	if err != nil {
		l.WithError(err).Errorf("Unable to write claim status for character [%d].", characterId)
		return
	}
	err = session.Announce(l)(ctx)(wp)(reportcb.ClaimAvailableTimeWriter)(writer.ClaimAvailableTimeBody(0, 0))(s)
	if err != nil {
		l.WithError(err).Errorf("Unable to write claim availability for character [%d].", characterId)
	}
}

func processStateReturn(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(accountId uint32, state uint8, params model2.SetField) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(accountId uint32, state uint8, params model2.SetField) model.Operator[session.Model] {
		sp := session.NewProcessor(l, ctx)
		return func(wp writer.Producer) func(accountId uint32, state uint8, params model2.SetField) model.Operator[session.Model] {
			return func(accountId uint32, state uint8, params model2.SetField) model.Operator[session.Model] {
				return func(s session.Model) error {
					if params.CharacterId <= 0 {
						return nil
					}

					cp := character.NewProcessor(l, ctx)
					c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator, cp.SkillModelDecorator, cp.QuestModelDecorator, cp.MonsterBookDecorator)(params.CharacterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to locate character [%d] attempting to login.", params.CharacterId)
						return sp.Destroy(s)
					}
					bl, err := buddylist.NewProcessor(l, ctx).GetById(params.CharacterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to locate buddylist [%d] attempting to login.", params.CharacterId)
						return sp.Destroy(s)
					}

					// Populate the couple/friendship ring cache once, here,
					// at the character-load path (task-269 task 12): this is
					// the single point every login AND channel-enter session
					// bootstrap runs through, so BuildCharacterData's and
					// CharacterSpawnBody's cache-only ring getters (PRD §8)
					// have data before their first encode without
					// re-fetching on every later map change, cash-shop open,
					// or ITC entry -- none of which call Populate again.
					// Fail-soft: Populate itself never returns an error (a
					// cashshop outage just leaves the cache empty).
					_ = ring.NewProcessor(l, ctx).Populate(c.Id())

					s = sp.SetAccountId(s.SessionId(), c.AccountId())
					s = sp.SetCharacterId(s.SessionId(), c.Id())
					s = sp.SetGm(s.SessionId(), c.Gm())

					f, lerr := location.GetField(l, ctx, c.Id())
					if lerr != nil {
						if errors.Is(lerr, location.ErrNotFound) {
							l.Errorf("Session bootstrap: no atlas-maps location found for [%d]; aborting (a session cannot bootstrap without a chosen map).", c.Id())
						} else {
							l.WithError(lerr).Errorf("Session bootstrap: atlas-maps unreachable for [%d] (infrastructure error); aborting.", c.Id())
						}
						return sp.Destroy(s)
					}
					s = sp.SetField(s.SessionId(), f)

					sp.SessionCreated(s)

					l.Debugf("Writing SetField for character [%d].", c.Id())
					err = session.Announce(l)(ctx)(wp)(fieldcb.SetFieldWriter)(writer.SetFieldBody(s.ChannelId(), c, bl))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to show set field response for character [%d]", c.Id())
					}
					// SpawnForSelf must be called synchronously after SetField so that the
					// client receives spawn packets in the correct order (SetField first).
					if serr := mapconsumer.SpawnForSelf(l, ctx, wp)(s, f); serr != nil {
						l.WithError(serr).Warnf("SpawnForSelf failed for character [%d] during session bootstrap; continuing.", c.Id())
					}
					routine.Go(l, ctx, func(_ context.Context) {
						entries := make([]buddyCB.BuddyEntry, 0, len(bl.Buddies()))
						for _, b := range bl.Buddies() {
							entries = append(entries, buddyCB.BuddyEntry{CharacterId: b.CharacterId(), Name: b.Name(), ChannelId: channel.Id(b.ChannelId()), Group: b.Group(), InShop: b.InShop()})
						}
						err := session.Announce(l)(ctx)(wp)(buddypkt.BuddyOperationWriter)(buddypkt.BuddyListUpdateBody(entries))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to write character [%d] buddy list.", c.Id())
						}
					})
					routine.Go(l, ctx, func(_ context.Context) {
						claimEnableAnnouncer(l, ctx, wp, s, c.Id())
					})
					routine.Go(l, ctx, func(_ context.Context) {
						g, _ := guild.NewProcessor(l, ctx).GetByMemberId(c.Id())
						if g.Id() != 0 {
							inGuild := g.Id() != 0
							var titles [5]string
							for _, t := range g.Titles() {
								idx := t.Index()
								if idx >= 1 && idx <= 5 {
									titles[idx-1] = t.Name()
								}
							}
							var guildMembers []guildcb.GuildMemberInfo
							for _, mm := range g.Members() {
								guildMembers = append(guildMembers, guildcb.GuildMemberInfo{
									CharacterId:   mm.CharacterId(),
									Name:          mm.Name(),
									JobId:         mm.JobId(),
									Level:         mm.Level(),
									Title:         mm.Title(),
									Online:        mm.Online(),
									Signature:     0,
									AllianceTitle: mm.AllianceTitle(),
								})
							}
							err = session.Announce(l)(ctx)(wp)(guildcb.GuildOperationWriter)(guildpkt.GuildInfoBody(inGuild, g.Id(), g.Name(), titles, guildMembers, g.Capacity(), g.LogoBackground(), g.LogoBackgroundColor(), g.Logo(), g.LogoColor(), g.Notice(), g.Points(), g.AllianceId()))(s)
							if err != nil {
								l.WithError(err).Errorf("Unable to write character [%d] buddy list.", c.Id())
							}
						}
					})
					routine.Go(l, ctx, func(_ context.Context) {
						var km map[int32]key.Model
						km, err = model.CollectToMap[key.Model, int32, key.Model](key.NewProcessor(l, ctx).ByCharacterIdProvider(s.CharacterId()), func(m key.Model) int32 {
							return m.Key()
						}, func(m key.Model) key.Model {
							return m
						})()
						if err != nil {
							l.WithError(err).Errorf("Unable to show key map for character [%d].", s.CharacterId())
							return
						}

						bindings := make(map[int32]charcb.KeyBinding)
						for k, v := range km {
							bindings[k] = charcb.KeyBinding{KeyType: v.Type(), KeyAction: v.Action()}
						}
						err = session.Announce(l)(ctx)(wp)(charcb.CharacterKeyMapWriter)(charcb.NewCharacterKeyMap(bindings).Encode)(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to show key map for character [%d].", s.CharacterId())
						}

						haction := int32(0)
						if hkm, ok := km[91]; ok {
							haction = hkm.Action()
						}
						err = session.Announce(l)(ctx)(wp)(charcb.CharacterKeyMapAutoHpWriter)(charcb.NewCharacterKeyMapAutoHp(haction).Encode)(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to show auto hp key map for character [%d].", s.CharacterId())
						}

						maction := int32(0)
						if mkm, ok := km[92]; ok {
							maction = mkm.Action()
						}
						err = session.Announce(l)(ctx)(wp)(charcb.CharacterKeyMapAutoMpWriter)(charcb.NewCharacterKeyMapAutoMp(maction).Encode)(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to show auto mp key map for character [%d].", s.CharacterId())
						}
					})
					routine.Go(l, ctx, func(_ context.Context) {
						var bs []buff.Model
						bs, err = buff.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
						if err != nil {
							l.WithError(err).Debugf("Unable to retrieve active buffs for character [%d].", s.CharacterId())
							return
						}
						// Mounts are transient across logins. Don't re-render a
						// persisted MONSTER_RIDING buff on login — cancel it so the player isn't
						// auto-remounted and the stale buff is cleared from atlas-buffs. The mount
						// progression (level/exp/tiredness) lives in atlas-mounts and is unaffected.
						rendered := make([]buff.Model, 0, len(bs))
						for _, b := range bs {
							if buff.IsMount(b) {
								if cerr := buff.NewProcessor(l, ctx).Cancel(s.Field(), s.CharacterId(), b.SourceId()); cerr != nil {
									l.WithError(cerr).Warnf("Unable to clear persisted mount buff for character [%d] on login.", s.CharacterId())
								}
								continue
							}
							rendered = append(rendered, b)
						}
						err = session.Announce(l)(ctx)(wp)(charcb.CharacterBuffGiveWriter)(writer.CharacterBuffGiveBody(rendered))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to write character [%d] buddy list.", c.Id())
						}
					})
					routine.Go(l, ctx, func(_ context.Context) {
						var w world.Model
						w, err = world.NewProcessor(l, ctx).GetById(s.WorldId())
						if err != nil {
							return
						}
						err = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessageTopScrollBody(w.Message()))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to write character [%d] buddy list.", c.Id())
						}
					})
					routine.Go(l, ctx, func(_ context.Context) {
						var sms []macro.Model
						sms, err = macro.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
						if err != nil {
							l.WithError(err).Errorf("Unable to read skill macros for character [%d].", c.Id())
							return
						}
						sort.Slice(sms, func(i, j int) bool {
							return sms[i].Id() < sms[j].Id()
						})
						ems := make([]charcb.SkillMacroEntry, 0)
						for _, sm := range sms {
							ems = append(ems, charcb.NewSkillMacroEntry(sm.Name(), sm.Shout(), sm.SkillId1(), sm.SkillId2(), sm.SkillId3()))
						}
						macros := charcb.NewSkillMacro(ems...)
						err = session.Announce(l)(ctx)(wp)(charcb.CharacterSkillMacroWriter)(macros.Encode)(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to show key map for character [%d].", s.CharacterId())
						}
					})
					routine.Go(l, ctx, func(_ context.Context) {
						var nms []note.Model
						nms, err = note.NewProcessor(l, ctx).GetByCharacter(s.CharacterId())
						if err != nil {
							l.WithError(err).Errorf("Unable to read notes for character [%d].", c.Id())
							return
						}
						if len(nms) == 0 {
							return
						}

						cnm := make(map[uint32]string)

						var wnms []model2.Note
						wnms, err = model.SliceMap(func(m note.Model) (model2.Note, error) {
							var sn string
							var ok bool
							if sn, ok = cnm[m.SenderId()]; !ok {
								c, err = character.NewProcessor(l, ctx).GetById()(m.SenderId())
								if err != nil {
									cnm[m.SenderId()] = "Unknown"
									sn = "Unknown"
								} else {
									cnm[m.SenderId()] = c.Name()
									sn = c.Name()
								}
							}

							return model2.Note{
								Id:         m.Id(),
								SenderName: sn,
								Message:    m.Message(),
								Timestamp:  m.Timestamp(),
								Flag:       m.Flag(),
							}, nil
						})(model.FixedProvider(nms))(model.ParallelMap())()

						noteEntries := make([]notepkt.NoteEntry, len(wnms))
						for i, n := range wnms {
							noteEntries[i] = notepkt.NoteEntry{Id: n.Id, SenderName: n.SenderName, Message: n.Message, Timestamp: n.Timestamp, Flag: n.Flag}
						}
						err = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteDisplayBody(noteEntries))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to show key map for character [%d].", s.CharacterId())
						}
					})
					return nil
				}
			}
		}
	}
}

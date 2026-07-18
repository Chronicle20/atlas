package buff

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	npc2 "atlas-channel/data/npc"
	consumer2 "atlas-channel/kafka/consumer"
	buff2 "atlas-channel/kafka/message/buff"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	controllernpc "atlas-channel/npc/controller"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_buff_status_event")(buff2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(buff2.EnvEventStatusTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventApplied(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExpired(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventGmHideApplied(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventGmHideExpired(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleStatusEventApplied(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffApplied {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			bs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			bs = append(bs, buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveWriter)(writer.CharacterBuffGiveBody(bs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
			}

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
				err = session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveForeignWriter)(writer.CharacterBuffGiveForeignBody(e.CharacterId, bs))(os)
				if err != nil {
					l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
					return err
				}
				return nil
			})
			return nil
		})
	}
}

func handleStatusEventExpired(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffExpired {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			ebs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			ebs = append(ebs, buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))

			err := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffCancelWriter)(writer.CharacterBuffCancelBody(ebs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write character [%d] cancelled buffs.", e.CharacterId)
			}

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
				err = session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffCancelForeignWriter)(writer.CharacterBuffCancelForeignBody(e.CharacterId, ebs))(os)
				if err != nil {
					l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
					return err
				}
				return nil
			})
			return nil
		})
	}
}

// handleStatusEventGmHideApplied relinquishes the hiding GM's NPCs
// (task-176, FR-6.1): revoke their client-side grants, then reassign to a
// visible session. Fires ONLY for SuperGmHide (9101004); Dark Sight and
// all other buffs are untouched.
func handleStatusEventGmHideApplied(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffApplied {
			return
		}
		if e.Body.SourceId != int32(skill2.SuperGmHideId) {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			f := s.Field()
			cp := controllernpc.NewProcessor(l, ctx)
			released, err := cp.ReleaseFor(f, s.CharacterId())
			if err != nil {
				l.WithError(err).Warnf("GM-hide: unable to release NPC controller entries for [%d] in field [%s].", s.CharacterId(), f.Id())
				return nil
			}
			if len(released) == 0 {
				l.Debugf("GM-hide: character [%d] controlled no NPCs in field [%s].", s.CharacterId(), f.Id())
				return nil
			}
			for _, npcId := range released {
				if rerr := controllernpc.AnnounceRevoke(l, ctx, wp)(s, npcId); rerr != nil {
					l.WithError(rerr).Warnf("GM-hide: unable to revoke NPC [%d] control from [%d].", npcId, s.CharacterId())
				}
			}
			assignments, aerr := cp.ElectFor(f, released, s.CharacterId())
			if aerr != nil {
				l.WithError(aerr).Warnf("GM-hide: unable to re-elect NPC controllers in field [%s].", f.Id())
				return nil
			}
			for npcId, winner := range assignments {
				if gerr := controllernpc.AnnounceGrant(l, ctx, wp)(f, winner, npcId); gerr != nil {
					l.WithError(gerr).Warnf("GM-hide: unable to announce NPC [%d] grant to [%d].", npcId, winner)
				}
			}
			l.Debugf("GM-hide: character [%d] relinquished [%d] NPCs in field [%s]; reassigned [%d].", s.CharacterId(), len(released), f.Id(), len(assignments))
			return nil
		})
	}
}

// handleStatusEventGmHideExpired restores the revealed GM's candidacy
// (FR-6.3): elect controllers for currently-uncontrolled NPCs with the GM
// back in the pool. No forced transfer — live controllers keep their NPCs.
// (atlas-buffs prunes its registry BEFORE emitting EXPIRED, so the
// winner-check cannot see a stale hide buff.)
func handleStatusEventGmHideExpired(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffExpired {
			return
		}
		if e.Body.SourceId != int32(skill2.SuperGmHideId) {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			f := s.Field()
			// Use InMapModelProvider + a sequential range rather than
			// ForEachInMap: the latter runs its callback via
			// model.ForEachSlice(..., model.ParallelExecute()) — one
			// goroutine per NPC — so accumulating into npcIds from inside
			// that callback would race on the shared slice header
			// (task-176 review; same bug class as the Task 9 hiddenCache
			// race fixed in e6c75ed42). InMapModelProvider(...)() fetches
			// the slice synchronously; the append loop below runs on this
			// goroutine only.
			npcs, err := npc2.NewProcessor(l, ctx).InMapModelProvider(f.MapId())()
			if err != nil {
				l.WithError(err).Warnf("GM-reveal: unable to enumerate NPCs in map [%d].", f.MapId())
				return nil
			}
			npcIds := make([]uint32, 0, len(npcs))
			for _, n := range npcs {
				npcIds = append(npcIds, n.Id())
			}
			cp := controllernpc.NewProcessor(l, ctx)
			unc, err := cp.UncontrolledIn(f, npcIds)
			if err != nil {
				l.WithError(err).Warnf("GM-reveal: unable to compute uncontrolled NPCs in field [%s].", f.Id())
				return nil
			}
			if len(unc) == 0 {
				return nil
			}
			assignments, aerr := cp.ElectFor(f, unc)
			if aerr != nil {
				l.WithError(aerr).Warnf("GM-reveal: unable to elect NPC controllers in field [%s].", f.Id())
				return nil
			}
			for npcId, winner := range assignments {
				if gerr := controllernpc.AnnounceGrant(l, ctx, wp)(f, winner, npcId); gerr != nil {
					l.WithError(gerr).Warnf("GM-reveal: unable to announce NPC [%d] grant to [%d].", npcId, winner)
				}
			}
			l.Debugf("GM-reveal: elected controllers for [%d] of [%d] uncontrolled NPCs in field [%s].", len(assignments), len(unc), f.Id())
			return nil
		})
	}
}

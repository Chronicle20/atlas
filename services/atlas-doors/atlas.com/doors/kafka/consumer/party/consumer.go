package party

import (
	mapdata "atlas-doors/data/map"
	enginedoor "atlas-doors/door"
	consumer2 "atlas-doors/kafka/consumer"
	"atlas-doors/party"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("party_status_event")(EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(EnvEventStatusTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleJoined(l)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLeft(l)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleExpel(l)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDisband(l)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeLeader(l)))); err != nil {
			return err
		}
		return nil
	}
}

// townPortalsForMap returns a closure that fetches door-type (Type==6) portals
// for a town map from atlas-data.  The closure is cheap to construct and
// evaluated lazily per-door, so one consumer event may trigger multiple fetches
// when party members have doors in different town maps.
func townPortalsForMap(l logrus.FieldLogger, ctx context.Context) func(_map.Id) []enginedoor.TownPortal {
	mp := mapdata.NewProcessor(l, ctx)
	return func(townMapId _map.Id) []enginedoor.TownPortal {
		const doorPortalType uint8 = 6
		m, err := mp.GetById(townMapId)
		if err != nil {
			return nil
		}
		portals := make([]enginedoor.TownPortal, 0)
		for _, p := range m.Portals() {
			if p.Type() == doorPortalType {
				portals = append(portals, enginedoor.TownPortal{X: p.X(), Y: p.Y()})
			}
		}
		return portals
	}
}

// handleJoined fires on JOINED events.  After a join the new member's slot is
// already reflected in the party returned by atlas-parties, so leavers is nil.
func handleJoined(l logrus.FieldLogger) message.Handler[StatusEvent[JoinedEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[JoinedEventBody]) {
		if e.Type != EventPartyStatusTypeJoined {
			return
		}
		pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Warnf("handleJoined: party %d not found", e.PartyId)
			return
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			pm.Members(), []character.Id{character.Id(e.ActorId)}, nil, townPortalsForMap(l, ctx))
	}
}

// handleLeft fires on LEFT events.  e.ActorId is the character who just left;
// the party returned by atlas-parties already reflects the post-leave member list.
// If the party is gone (solo leave that disbanded it), members is nil and the
// leaver's door is still dropped to solo by ReconcileParty.
func handleLeft(l logrus.FieldLogger) message.Handler[StatusEvent[LeftEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[LeftEventBody]) {
		if e.Type != EventPartyStatusTypeLeft {
			return
		}
		var members []character.Id
		if pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId); err == nil {
			members = pm.Members()
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			members, nil, []character.Id{character.Id(e.ActorId)}, townPortalsForMap(l, ctx))
	}
}

// handleExpel fires on EXPEL events.  e.Body.CharacterId is the expelled member.
func handleExpel(l logrus.FieldLogger) message.Handler[StatusEvent[ExpelEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[ExpelEventBody]) {
		if e.Type != EventPartyStatusTypeExpel {
			return
		}
		var members []character.Id
		if pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId); err == nil {
			members = pm.Members()
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			members, nil, []character.Id{character.Id(e.Body.CharacterId)}, townPortalsForMap(l, ctx))
	}
}

// handleDisband fires on DISBAND events.  The party no longer exists in
// atlas-parties; the event body carries the full former member list (including
// the departing leader, after the Task 1 fix).  Every member's door drops to
// solo via ReconcileParty(members=nil, leavers=formerMembers).
func handleDisband(l logrus.FieldLogger) message.Handler[StatusEvent[DisbandEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[DisbandEventBody]) {
		if e.Type != EventPartyStatusTypeDisband {
			return
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			nil, nil, e.Body.Members, townPortalsForMap(l, ctx))
	}
}

// handleChangeLeader fires on CHANGE_LEADER events.  The leader slot (index 0)
// is now a different character, so the ordering may have changed.  No one left
// the party, so joiners and leavers are both nil.
func handleChangeLeader(l logrus.FieldLogger) message.Handler[StatusEvent[ChangeLeaderEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[ChangeLeaderEventBody]) {
		if e.Type != EventPartyStatusTypeChangeLeader {
			return
		}
		pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Warnf("handleChangeLeader: party %d not found", e.PartyId)
			return
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			pm.Members(), nil, nil, townPortalsForMap(l, ctx))
	}
}

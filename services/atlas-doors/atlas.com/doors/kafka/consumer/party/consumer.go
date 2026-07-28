package party

import (
	mapdata "atlas-doors/data/map"
	enginedoor "atlas-doors/door"
	consumer2 "atlas-doors/kafka/consumer"
	"atlas-doors/party"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// On a party membership change the door service reslots each affected member's
// door TOWN-portal position to the member's current party slot. This is the
// town side only — it never re-sends the area door, so it cannot toggle the
// client render (the source of the earlier below-platform / expiry-crash bugs).
// The reslot keeps the door's stored town position correct, which is both the
// in-town render position and the warp destination when the door is entered.
// (Two members' doors must warp to portal index 0 and 1, not both to 0.)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("party_status_event")(EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		handlers := []handler.Handler{
			message.AdaptHandler(message.PersistentConfig(handleJoined(l))),
			message.AdaptHandler(message.PersistentConfig(handleLeft(l))),
			message.AdaptHandler(message.PersistentConfig(handleExpel(l))),
			message.AdaptHandler(message.PersistentConfig(handleDisband(l))),
			message.AdaptHandler(message.PersistentConfig(handleChangeLeader(l))),
		}
		for _, h := range handlers {
			if _, err := rf(t, h); err != nil {
				return err
			}
		}
		return nil
	}
}

// townPortalsForMap returns a closure that fetches door-type (Type==6) portals
// for a town map from atlas-data, used to resolve a slot's town-portal position.
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

// reslotForParty resolves the party's current ordered membership and reslots the
// members' (and leavers') doors. members is nil on disband (party gone).
func reslotForParty(l logrus.FieldLogger, ctx context.Context, partyId uint32, members []character.Id, leavers []character.Id) {
	_ = enginedoor.ReslotParty(enginedoor.NewProcessor(l, ctx), partyId, members, leavers, townPortalsForMap(l, ctx))
}

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
		reslotForParty(l, ctx, e.PartyId, pm.Members(), nil)
	}
}

func handleLeft(l logrus.FieldLogger) message.Handler[StatusEvent[LeftEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[LeftEventBody]) {
		if e.Type != EventPartyStatusTypeLeft {
			return
		}
		var members []character.Id
		if pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId); err == nil {
			members = pm.Members()
		}
		reslotForParty(l, ctx, e.PartyId, members, []character.Id{e.ActorId})
	}
}

func handleExpel(l logrus.FieldLogger) message.Handler[StatusEvent[ExpelEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[ExpelEventBody]) {
		if e.Type != EventPartyStatusTypeExpel {
			return
		}
		var members []character.Id
		if pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId); err == nil {
			members = pm.Members()
		}
		reslotForParty(l, ctx, e.PartyId, members, []character.Id{e.Body.CharacterId})
	}
}

func handleDisband(l logrus.FieldLogger) message.Handler[StatusEvent[DisbandEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[DisbandEventBody]) {
		if e.Type != EventPartyStatusTypeDisband {
			return
		}
		reslotForParty(l, ctx, e.PartyId, nil, e.Body.Members)
	}
}

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
		reslotForParty(l, ctx, e.PartyId, pm.Members(), nil)
	}
}

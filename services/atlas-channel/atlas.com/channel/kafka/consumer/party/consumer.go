package party

import (
	"atlas-channel/character"
	"atlas-channel/door"
	consumer2 "atlas-channel/kafka/consumer"
	party2 "atlas-channel/kafka/message/party"
	"atlas-channel/listener"
	"atlas-channel/maps/location"
	"atlas-channel/party"
	"atlas-channel/party/hpsync"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	partypkt "github.com/Chronicle20/atlas/libs/atlas-packet/party"
	partycb "github.com/Chronicle20/atlas/libs/atlas-packet/party/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func toPartyMembers(l logrus.FieldLogger, ctx context.Context, p party.Model, forChannel channel.Id) []partypkt.PartyMember {
	members := make([]partypkt.PartyMember, 0, len(p.Members()))
	dp := door.NewProcessor(l, ctx)
	for _, m := range p.Members() {
		chId := int32(m.ChannelId())
		if !m.Online() {
			chId = -2
		}
		mapId := uint32(0)
		if forChannel == m.ChannelId() {
			mapId = uint32(m.MapId())
		}
		pm := partypkt.PartyMember{
			Id:        m.Id(),
			Name:      m.Name(),
			JobId:     uint16(m.JobId()),
			Level:     uint16(m.Level()),
			ChannelId: chId,
			MapId:     mapId,
		}
		applyMemberDoor(&pm, dp, m.Id())
		members = append(members, pm)
	}
	return members
}

// applyMemberDoor populates the member's aTownPortal entry from their live
// Mystic Door (if any). The town-portal array is how the v83 client renders
// party-member doors in town — a doorless member keeps the zero entry. The
// portal carries the town (exit) map, the area (origin) map, and the AREA-side
// door position (matching SPAWN_PORTAL / the v83 client partyPortal toPosition()).
func applyMemberDoor(pm *partypkt.PartyMember, dp *door.Processor, memberId uint32) {
	doors, err := dp.GetByOwner(memberId)
	if err != nil || len(doors) == 0 {
		return
	}
	d := doors[0]
	pm.HasDoor = true
	pm.DoorTownMapId = uint32(d.TownMapId())
	pm.DoorFieldMapId = uint32(d.Field().MapId())
	pm.DoorX = d.AreaX()
	pm.DoorY = d.AreaY()
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("party_status_event")(party2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(party2.EnvEventStatusTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleLeft(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleExpel(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleDisband(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleJoin(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleChangeLeader(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleError(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleCreated(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.CreatedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.CreatedEventBody]) {
		if e.Type != party2.EventPartyStatusTypeCreated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Warnf("Received created event for party [%d] which does not exist.", e.PartyId)
			return
		}

		// Resolve the party leader's active Mystic Door (if any) so that the
		// party-created packet can populate the minimap door indicator (FR-3.3).
		// the v83 client partyCreated convention: townMapId = door destination (town),
		// targetMapId = door origin (area/dungeon), x/y = area-side door position.
		townMapId := _map.EmptyMapId
		targetMapId := _map.EmptyMapId
		var doorX, doorY int16
		doors, derr := door.NewProcessor(l, ctx).GetByOwner(p.LeaderId())
		if derr != nil {
			l.WithError(derr).Warnf("Unable to retrieve doors for party leader [%d]; sending empty sentinel.", p.LeaderId())
		} else if len(doors) > 0 {
			d := doors[0]
			townMapId = d.TownMapId()
			targetMapId = d.Field().MapId()
			doorX = d.AreaX()
			doorY = d.AreaY()
		}

		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(p.LeaderId(), partyCreated(l)(ctx)(wp)(e.PartyId, townMapId, targetMapId, doorX, doorY))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce party [%d] created to character [%d].", e.PartyId, p.LeaderId())
		}
	}
}

func partyCreated(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, townMapId _map.Id, targetMapId _map.Id, doorX int16, doorY int16) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, townMapId _map.Id, targetMapId _map.Id, doorX int16, doorY int16) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, townMapId _map.Id, targetMapId _map.Id, doorX int16, doorY int16) model.Operator[session.Model] {
			return func(partyId uint32, townMapId _map.Id, targetMapId _map.Id, doorX int16, doorY int16) model.Operator[session.Model] {
				if townMapId == _map.EmptyMapId {
					return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyCreatedBody(partyId))
				}
				return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyCreatedBodyWithDoor(partyId, townMapId, targetMapId, doorX, doorY))
			}
		}
	}
}

func handleLeft(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.LeftEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.LeftEventBody]) {
		if e.Type != party2.EventPartyStatusTypeLeft {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for character [%d] which does not exist.", e.ActorId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range p.Members() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyLeft(l)(ctx)(wp)(p, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] has left party [%d].", tc.Id(), p.Id())
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ActorId, partyLeft(l)(ctx)(wp)(p, tc, sc.ChannelId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] has left party [%d].", tc.Id(), p.Id())
			}
		}()

	}
}

func partyLeft(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
			return func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyLeftBody(p.Id(), tc.Id(), tc.Name(), toPartyMembers(l, ctx, p, forChannel), p.LeaderId()))
			}
		}
	}
}

func handleExpel(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.ExpelEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.ExpelEventBody]) {
		if e.Type != party2.EventPartyStatusTypeExpel {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received expel event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Received expel event for character [%d] which does not exist.", e.Body.CharacterId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range p.Members() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyExpel(l)(ctx)(wp)(p, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] was expelled from party [%d].", tc.Id(), p.Id())
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, partyExpel(l)(ctx)(wp)(p, tc, sc.ChannelId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] was expelled from party [%d].", tc.Id(), p.Id())
			}
		}()

	}
}

func partyExpel(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
			return func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyExpelBody(p.Id(), tc.Id(), tc.Name(), toPartyMembers(l, ctx, p, forChannel), p.LeaderId()))
			}
		}
	}
}

func handleDisband(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.DisbandEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.DisbandEventBody]) {
		if e.Type != party2.EventPartyStatusTypeDisband {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received disband event for character [%d] which does not exist.", e.ActorId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range e.Body.Members {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m, partyDisband(l)(ctx)(wp)(e.PartyId, tc, sc.ChannelId()))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] the party [%d] was disbanded.", m, e.PartyId)
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ActorId, partyDisband(l)(ctx)(wp)(e.PartyId, tc, sc.ChannelId()))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] the party [%d] was disbanded.", e.ActorId, e.PartyId)
			}
		}()

	}
}

func partyDisband(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
			return func(partyId uint32, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyDisbandBody(partyId, tc.Id()))
			}
		}
	}
}

func handleJoin(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.JoinedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.JoinedEventBody]) {
		if e.Type != party2.EventPartyStatusTypeJoined {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for party [%d] which does not exist.", e.PartyId)
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received join event for character [%d] which does not exist.", e.ActorId)
			return
		}

		// Announce the join to every party member synchronously FIRST. The v83
		// PARTYDATA carries no HP, so this join packet sets every party gauge to 0;
		// the hpsync below must reach each session's writer AFTER it, or the join
		// packet overwrites the synced gauges back to 0. The previous code raced
		// these in separate goroutines, so the joiner's HP frequently stuck at 0.
		for _, m := range p.Members() {
			if aErr := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyJoined(l)(ctx)(wp)(p, tc, sc.ChannelId())); aErr != nil {
				l.WithError(aErr).Errorf("Unable to announce party [%d] joined party [%d].", e.PartyId, p.Id())
			}
		}

		// Sync the joining character's HP gauges in both directions with the in-map
		// party members (PARTYDATA has no HP). Enqueued after the join announces
		// above so it is not overwritten. Done only on the actor's own channel
		// server to avoid redundant cross-channel work.
		if f, ferr := location.GetField(l, ctx, e.ActorId); ferr != nil {
			l.WithError(ferr).Debugf("Unable to resolve field for character [%d]; skipping party member HP sync on join.", e.ActorId)
		} else if f.ChannelId() == sc.ChannelId() {
			if hpErr := hpsync.Sync(l, ctx, wp, f, e.ActorId); hpErr != nil {
				l.WithError(hpErr).Debugf("Unable to sync party member HP for character [%d] on join.", e.ActorId)
			}
		}
	}
}

func partyJoined(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
			return func(p party.Model, tc character.Model, forChannel channel.Id) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyJoinBody(p.Id(), tc.Name(), toPartyMembers(l, ctx, p, forChannel), p.LeaderId()))
			}
		}
	}
}

func handleChangeLeader(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.ChangeLeaderEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.ChangeLeaderEventBody]) {
		if e.Type != party2.EventPartyStatusTypeChangeLeader {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		p, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Errorf("Received expel event for party [%d] which does not exist.", e.PartyId)
			return
		}

		// For remaining party members.
		go func() {
			for _, m := range p.Members() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), partyChangeLeader(l)(ctx)(wp)(e.PartyId, e.Body.CharacterId, e.Body.Disconnected))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce change party [%d] leadership to [%d].", e.PartyId, e.Body.CharacterId)
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, partyChangeLeader(l)(ctx)(wp)(e.PartyId, e.Body.CharacterId, e.Body.Disconnected))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce change party [%d] leadership to [%d].", e.PartyId, e.Body.CharacterId)
			}
		}()

	}
}

func partyChangeLeader(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, targetCharacterId uint32, disconnected bool) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(partyId uint32, targetCharacterId uint32, disconnected bool) model.Operator[session.Model] {
		return func(wp writer.Producer) func(partyId uint32, targetCharacterId uint32, disconnected bool) model.Operator[session.Model] {
			return func(partyId uint32, targetCharacterId uint32, disconnected bool) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyChangeLeaderBody(targetCharacterId, disconnected))
			}
		}
	}
}

func handleError(sc server.Model, wp writer.Producer) message.Handler[party2.StatusEvent[party2.ErrorEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.ErrorEventBody]) {
		if e.Type != party2.EventPartyStatusTypeError {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ActorId, partyError(l)(ctx)(wp)(e.Body.Type, e.Body.CharacterName))
	}
}

func partyError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(errorType string, name string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(errorType string, name string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(errorType string, name string) model.Operator[session.Model] {
			return func(errorType string, name string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyErrorBody(errorType, name))
			}
		}
	}
}

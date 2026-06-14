package door

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	doorcb "github.com/Chronicle20/atlas/libs/atlas-packet/door/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("door_status_event")(EnvEventTopicDoorStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(EnvEventTopicDoorStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoved(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSlotChanged(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// partyMemberSet resolves the set of character ids eligible to see a door: the
// owner (always included) plus every same-channel party member of the owner.
// Held as a package-level var so tests can stub party membership without a REST
// mock. partyId == 0 (no party) yields just the owner.
var partyMemberSet = func(l logrus.FieldLogger, ctx context.Context, ownerCharacterId, partyId uint32) map[uint32]struct{} {
	members := map[uint32]struct{}{ownerCharacterId: {}}
	if partyId == 0 {
		return members
	}
	p, err := party.NewProcessor(l, ctx).GetById(partyId)
	if err != nil {
		l.WithError(err).Warnf("Unable to resolve party [%d] for door owner [%d]; restricting to owner only.", partyId, ownerCharacterId)
		return members
	}
	for _, m := range p.Members() {
		members[m.Id()] = struct{}{}
	}
	return members
}

// broadcastDoorToEligible announces `enc` (writer `writerName`) to the sessions
// in field `f` whose character is the owner or a same-channel party member of
// the owner (caster always included). Held as a package-level var so the test
// can stub session enumeration + party membership.
var broadcastDoorToEligible = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, ownerCharacterId, partyId uint32, writerName string, enc packet.Encode) {
	members := partyMemberSet(l, ctx, ownerCharacterId, partyId)
	err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
		if _, ok := members[s.CharacterId()]; !ok {
			return nil
		}
		return session.Announce(l)(ctx)(wp)(writerName)(enc)(s)
	})
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast door packet [%s] to field [%d].", writerName, f.MapId())
	}
}

func handleCreated(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[CreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[CreatedBody]) {
		if e.Type != EventDoorStatusCreated {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		b := e.Body
		areaField := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		townField := field.NewBuilder(e.WorldId, e.ChannelId, b.TownMapId).SetInstance(e.Instance).Build()

		// AREA field viewers (Cosmic DoorObject.sendSpawnData, areaDoor: from=area,
		// inTown=false): spawnPortal(area, town, townPortalPos) then
		// spawnDoor(owner, areaPos, launched=false-for-first-deploy).
		broadcastDoorToEligible(l, ctx, wp, areaField, e.OwnerCharacterId, e.PartyId,
			doorcb.SpawnPortalWriter, writer.SpawnPortalBody(e.MapId, b.TownMapId, b.TownX, b.TownY))
		broadcastDoorToEligible(l, ctx, wp, areaField, e.OwnerCharacterId, e.PartyId,
			doorcb.SpawnDoorWriter, writer.SpawnDoorBody(e.OwnerCharacterId, b.AreaX, b.AreaY, false))

		// TOWN field viewers (Cosmic townDoor: from=town, inTown=true): ONLY
		// spawnPortal(town, area, areaPos) — NO spawnDoor (line 120 guards
		// spawnDoor behind !inTown()).
		broadcastDoorToEligible(l, ctx, wp, townField, e.OwnerCharacterId, e.PartyId,
			doorcb.SpawnPortalWriter, writer.SpawnPortalBody(b.TownMapId, e.MapId, b.AreaX, b.AreaY))
	}
}

func handleRemoved(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[RemovedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[RemovedBody]) {
		if e.Type != EventDoorStatusRemoved {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		b := e.Body
		areaField := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		townField := field.NewBuilder(e.WorldId, e.ChannelId, b.TownMapId).SetInstance(e.Instance).Build()

		// AREA viewers (areaDoor.sendDestroyData, inTown=false): removeDoor(owner).
		broadcastDoorToEligible(l, ctx, wp, areaField, e.OwnerCharacterId, e.PartyId,
			doorcb.RemoveDoorWriter, writer.RemoveDoorBody(e.OwnerCharacterId))

		// TOWN viewers (townDoor.sendDestroyData, inTown=true): removeDoor town=true
		// -> 8-byte SPAWN_PORTAL clear (RemoveTownDoor), NOT SpawnPortal(...,0,0).
		broadcastDoorToEligible(l, ctx, wp, townField, e.OwnerCharacterId, e.PartyId,
			doorcb.RemoveTownDoorWriter, writer.RemoveTownDoorBody())
	}
}

func handleSlotChanged(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[SlotChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[SlotChangedBody]) {
		if e.Type != EventDoorStatusSlotChanged {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		b := e.Body
		townField := field.NewBuilder(e.WorldId, e.ChannelId, b.TownMapId).SetInstance(e.Instance).Build()

		// Reslot moves only the TOWN-side minimap portal indicator (Cosmic
		// Door.updateDoorPortal updates the areaDoor's linked town portal; the
		// town portal indicator is re-placed at the new slot). Cosmic has no
		// dedicated reslot packet, so emit remove(town) + spawnPortal at the new
		// slot for the town field. The party-packet minimap update is Task G6.
		broadcastDoorToEligible(l, ctx, wp, townField, e.OwnerCharacterId, e.PartyId,
			doorcb.RemoveTownDoorWriter, writer.RemoveTownDoorBody())
		broadcastDoorToEligible(l, ctx, wp, townField, e.OwnerCharacterId, e.PartyId,
			doorcb.SpawnPortalWriter, writer.SpawnPortalBody(b.TownMapId, e.MapId, b.TownX, b.TownY))
	}
}

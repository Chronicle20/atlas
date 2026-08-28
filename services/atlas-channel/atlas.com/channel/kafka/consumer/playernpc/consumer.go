package playernpc

import (
	_map "atlas-channel/kafka/consumer/map"
	"atlas-channel/listener"
	mapProcessor "atlas-channel/map"
	controllernpc "atlas-channel/npc/controller"
	"atlas-channel/playernpc"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	consumer2 "atlas-channel/kafka/consumer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	mapc "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// announce is the session.Announce seam for this consumer's packets,
// package-level var (mirrors kafka/consumer/door's doorAnnounce and
// kafka/consumer/map's playerNpcAnnounce) so tests can stub it without a
// real socket writer.
var announce = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, writerName string, enc packet.Encode, s session.Model) error {
	return session.Announce(l)(ctx)(wp)(writerName)(enc)(s)
}

// announceOp adapts announce into a model.Operator[session.Model] for
// ForSessionsInMap.
func announceOp(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, writerName string, enc packet.Encode) model2.Operator[session.Model] {
	return func(s session.Model) error {
		return announce(l, ctx, wp, writerName, enc, s)
	}
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("player_npc_status_event")(EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(EnvEventTopicStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDeployed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleUpdated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoved(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRepositioned(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// toPlayerNpcModel converts a StatusModel (the DEPLOYED/UPDATED wire body)
// into playernpc.Model via the existing RestModel/Extract path, so the
// spawn packet construction (kafka/consumer/map.PlayerNpcSpawn /
// PlayerNpcImitatedEntry) has exactly one implementation shared by the
// map-enter replay path and this consumer.
func toPlayerNpcModel(m StatusModel) (playernpc.Model, error) {
	equipment := make([]playernpc.EquipmentRestModel, 0, len(m.Equipment))
	for _, e := range m.Equipment {
		equipment = append(equipment, playernpc.EquipmentRestModel{Slot: e.Slot, ItemId: e.ItemId})
	}
	rm := playernpc.RestModel{
		Id:             m.Id,
		CharacterId:    m.CharacterId,
		Name:           m.Name,
		WorldId:        world.Id(m.WorldId),
		MapId:          mapc.Id(m.MapId),
		ScriptId:       m.ScriptId,
		ObjectId:       m.ObjectId,
		Gender:         m.Gender,
		Skin:           m.Skin,
		Face:           m.Face,
		Hair:           m.Hair,
		JobId:          job.Id(m.JobId),
		X:              m.X,
		Cy:             m.Cy,
		Fh:             m.Fh,
		Rx0:            m.Rx0,
		Rx1:            m.Rx1,
		Dir:            m.Dir,
		WorldRank:      m.WorldRank,
		OverallRank:    m.OverallRank,
		WorldJobRank:   m.WorldJobRank,
		OverallJobRank: m.OverallJobRank,
		Equipment:      equipment,
		DeployedAt:     m.DeployedAt,
	}
	return playernpc.Extract(rm)
}

// broadcastSpawn sends the plain SPAWN_NPC (design D-4 -- no controller
// grant, FR-7.4) then the single-entry IMITATED_NPC_DATA for one deployed
// Player NPC to every session in field f -- this channel pod's own
// sessions only. Every channel of the world runs its own atlas-channel
// pod consuming the same broadcast event, so "every channel of the world"
// (plan.md Task 19) falls out of each pod independently broadcasting
// within its own field; there is no per-event channelId to gate on
// (kafka.go's StatusEvent carries none).
func broadcastSpawn(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, n playernpc.Model) {
	spawn := _map.PlayerNpcSpawn(n)
	entry := _map.PlayerNpcImitatedEntry(n)
	err := mapProcessor.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
		if err := announce(l, ctx, wp, npcpkt.NpcSpawnWriter, spawn.Encode, s); err != nil {
			return err
		}
		return announce(l, ctx, wp, npcpkt.NpcImitatedDataWriter, npcpkt.NewImitatedNpcData([]npcpkt.ImitatedNpc{entry}).Encode, s)
	})
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast Player NPC [%d] spawn to map [%d].", n.ObjectId(), f.MapId())
	}
	electController(l, ctx, wp, f, n.ObjectId())
}

// electController assigns one live session in f control of npcObjectId and
// announces the grant. Without it a Player NPC deployed while players are
// already standing in the map stays mute until someone re-enters, because
// CNpc::SetActive -- and therefore the client's chat balloon -- is reached
// only through the controller grant (task-251 bug report §5).
func electController(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, npcObjectId uint32) {
	cp := controllernpc.NewProcessor(l, ctx)
	assignments, err := cp.ElectFor(f, []uint32{npcObjectId})
	if err != nil {
		l.WithError(err).Warnf("Unable to elect a controller for Player NPC [%d] in map [%d].", npcObjectId, f.MapId())
		return
	}
	for npcId, winner := range assignments {
		if gerr := controllernpc.AnnounceGrant(l, ctx, wp)(f, winner, npcId); gerr != nil {
			l.WithError(gerr).Warnf("Unable to announce Player NPC [%d] controller grant to [%d].", npcId, winner)
		}
	}
}

// broadcastImitatedOnly sends IMITATED_NPC_DATA alone -- no despawn/respawn
// (UPDATED: the object has not moved, only its avatar refreshed).
func broadcastImitatedOnly(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, n playernpc.Model) {
	entry := _map.PlayerNpcImitatedEntry(n)
	err := mapProcessor.NewProcessor(l, ctx).ForSessionsInMap(f, announceOp(l, ctx, wp, npcpkt.NpcImitatedDataWriter, npcpkt.NewImitatedNpcData([]npcpkt.ImitatedNpc{entry}).Encode))
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast Player NPC [%d] update to map [%d].", n.ObjectId(), f.MapId())
	}
}

func handleDeployed(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[StatusModel]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusModel]) {
		if e.Type != EventTypeDeployed {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.Body.WorldId)) {
			return
		}
		n, err := toPlayerNpcModel(e.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to interpret DEPLOYED Player NPC [%s] body.", e.Body.Id)
			return
		}
		f := field.NewBuilder(world.Id(e.Body.WorldId), sc.ChannelId(), mapc.Id(e.Body.MapId)).Build()
		l.Debugf("Player NPC [%d] deployed in map [%d].", n.ObjectId(), f.MapId())
		broadcastSpawn(l, ctx, wp, f, n)
	}
}

func handleUpdated(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[StatusModel]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusModel]) {
		if e.Type != EventTypeUpdated {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.Body.WorldId)) {
			return
		}
		n, err := toPlayerNpcModel(e.Body)
		if err != nil {
			l.WithError(err).Errorf("Unable to interpret UPDATED Player NPC [%s] body.", e.Body.Id)
			return
		}
		f := field.NewBuilder(world.Id(e.Body.WorldId), sc.ChannelId(), mapc.Id(e.Body.MapId)).Build()
		l.Debugf("Player NPC [%d] updated in map [%d].", n.ObjectId(), f.MapId())
		broadcastImitatedOnly(l, ctx, wp, f, n)
	}
}

func handleRemoved(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[StatusRemovedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusRemovedBody]) {
		if e.Type != EventTypeRemoved {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.Body.WorldId)) {
			return
		}
		f := field.NewBuilder(world.Id(e.Body.WorldId), sc.ChannelId(), mapc.Id(e.Body.MapId)).Build()
		l.Debugf("Player NPC [%d] removed from map [%d].", e.Body.ObjectId, f.MapId())
		err := mapProcessor.NewProcessor(l, ctx).ForSessionsInMap(f, announceOp(l, ctx, wp, npcpkt.NpcRemoveWriter, npcpkt.NewNpcRemove(e.Body.ObjectId).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast Player NPC [%d] removal to map [%d].", e.Body.ObjectId, f.MapId())
		}
		// The object is gone; drop its controller entry so the oid cannot be
		// re-elected (and so a redeploy reusing the same script id claims
		// cleanly rather than inheriting a dead controller).
		if rerr := controllernpc.NewProcessor(l, ctx).Release(f, e.Body.ObjectId); rerr != nil {
			l.WithError(rerr).Warnf("Unable to release controller entry for removed Player NPC [%d].", e.Body.ObjectId)
		}
	}
}

// handleRepositioned despawns and respawns every listed object (design
// §5.4/§7.4 -- "never leave a client holding a stale position"), then
// re-sends one batched IMITATED_NPC_DATA for the whole list: the
// respawned objects are freshly materialized SPAWN_NPCs with no avatar
// attached yet, and REPOSITIONED's own body carries only positions, not
// appearance, so the current appearance is read back from
// atlas-player-npcs (the same map-enter read client Task 18 added) rather
// than re-derived here.
func handleRepositioned(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[StatusRepositionedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[StatusRepositionedBody]) {
		if e.Type != EventTypeRepositioned {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.Body.WorldId)) {
			return
		}
		if len(e.Body.Npcs) == 0 {
			return
		}
		f := field.NewBuilder(world.Id(e.Body.WorldId), sc.ChannelId(), mapc.Id(e.Body.MapId)).Build()

		current, err := playernpc.NewProcessor(l, ctx).InMapModelProvider(f)()
		if err != nil {
			l.WithError(err).Errorf("Unable to read current Player NPCs for map [%d] to service REPOSITIONED.", f.MapId())
			return
		}
		byObjectId := make(map[uint32]playernpc.Model, len(current))
		for _, n := range current {
			byObjectId[n.ObjectId()] = n
		}

		entries := make([]npcpkt.ImitatedNpc, 0, len(e.Body.Npcs))
		respawned := make([]uint32, 0, len(e.Body.Npcs))
		for _, rn := range e.Body.Npcs {
			n, ok := byObjectId[rn.ObjectId]
			if !ok {
				l.Warnf("REPOSITIONED referenced Player NPC object [%d] not currently deployed in map [%d]; skipping.", rn.ObjectId, f.MapId())
				continue
			}
			removeErr := mapProcessor.NewProcessor(l, ctx).ForSessionsInMap(f, announceOp(l, ctx, wp, npcpkt.NpcRemoveWriter, npcpkt.NewNpcRemove(rn.ObjectId).Encode))
			if removeErr != nil {
				l.WithError(removeErr).Errorf("Unable to broadcast Player NPC [%d] despawn for reposition to map [%d].", rn.ObjectId, f.MapId())
				continue
			}
			spawn := npcpkt.NewNpcSpawn(n.ObjectId(), n.ScriptId(), rn.X, rn.Cy, int32(n.Dir()), rn.Fh, rn.Rx0, rn.Rx1)
			spawnErr := mapProcessor.NewProcessor(l, ctx).ForSessionsInMap(f, announceOp(l, ctx, wp, npcpkt.NpcSpawnWriter, spawn.Encode))
			if spawnErr != nil {
				l.WithError(spawnErr).Errorf("Unable to broadcast Player NPC [%d] respawn for reposition to map [%d].", rn.ObjectId, f.MapId())
				continue
			}
			entries = append(entries, _map.PlayerNpcImitatedEntry(n))
			respawned = append(respawned, rn.ObjectId)
		}
		if len(entries) == 0 {
			return
		}
		err = mapProcessor.NewProcessor(l, ctx).ForSessionsInMap(f, announceOp(l, ctx, wp, npcpkt.NpcImitatedDataWriter, npcpkt.NewImitatedNpcData(entries).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast repositioned Player NPC avatar data to map [%d].", f.MapId())
		}

		// The despawn/respawn above destroyed and re-created each object
		// client-side, so every client's copy is back to inactive and the
		// old controller entry is stale. Release, then elect afresh, or the
		// repositioned NPCs go mute (task-251 bug report §5).
		cp := controllernpc.NewProcessor(l, ctx)
		if rerr := cp.Release(f, respawned...); rerr != nil {
			l.WithError(rerr).Warnf("Unable to release controller entries for repositioned Player NPCs in map [%d].", f.MapId())
			return
		}
		for _, objectId := range respawned {
			electController(l, ctx, wp, f, objectId)
		}
	}
}

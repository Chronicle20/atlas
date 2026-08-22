package _map

import (
	"atlas-channel/playernpc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// playerNpcAnnounce is the session.Announce seam for Player NPC packets,
// extracted as a package-level var (mirrors doorAnnounce) so tests can stub
// it without a real socket writer.
var playerNpcAnnounce = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, writerName string, enc packet.Encode, s session.Model) error {
	return session.Announce(l)(ctx)(wp)(writerName)(enc)(s)
}

// PlayerNpcSpawn builds the plain SPAWN_NPC packet for a deployed Player
// NPC n (design D-4). template is n's scriptId — the imitate-eligible WZ
// NPC template the client materializes the object from, before
// IMITATED_NPC_DATA overlays the frozen player appearance on it. dir is
// carried the same way data/npc.Model.F() is: 0/1 flips facing.
func PlayerNpcSpawn(n playernpc.Model) npcpkt.Spawn {
	return npcpkt.NewNpcSpawn(n.ObjectId(), n.ScriptId(), n.X(), n.Cy(), int32(n.Dir()), n.Fh(), n.RX0(), n.RX1())
}

// PlayerNpcAvatar converts a Player NPC's frozen appearance into the
// IMITATED_NPC_DATA avatar arm. A Player NPC has no masked (cash-covered)
// equipment slots or pets recorded (§4.3/§4.5 do not carry them), so both
// are empty.
func PlayerNpcAvatar(n playernpc.Model) packetmodel.Avatar {
	equips := make(map[slot.Position]uint32, len(n.Equipment()))
	for _, e := range n.Equipment() {
		equips[slot.Position(e.Slot())] = e.ItemId()
	}
	return packetmodel.NewAvatar(n.Gender(), n.Skin(), n.Face(), false, n.Hair(), equips, map[slot.Position]uint32{}, map[int8]uint32{})
}

// PlayerNpcImitatedEntry builds one IMITATED_NPC_DATA entry for a deployed
// Player NPC n. Its "templateId" field carries n's scriptId (design §7.1's
// avatar arm leads with scriptId, not a WZ NPC template).
func PlayerNpcImitatedEntry(n playernpc.Model) npcpkt.ImitatedNpc {
	return npcpkt.NewImitatedNpc(n.ScriptId(), n.Name(), PlayerNpcAvatar(n))
}

// spawnPlayerNpcsForSession sends plain SPAWN_NPC (no controller grant --
// design D-4/FR-7.4) for every Player NPC deployed in f to the entering
// session s, then one batched IMITATED_NPC_DATA carrying every spawned
// NPC's avatar data. Ordering is SPAWN_NPC-then-IMITATED_NPC_DATA always
// (FR-7.1): the client needs the object in its pool before the avatar data
// can attach to it.
//
// Sequential, not model.ParallelExecute()-fanned like the ordinary-NPC
// sibling (spawnNPCForSession): every SpawnNPC in the map must land before
// the one ImitatedNpcData that batches them, so the entries have to be
// collected deterministically rather than raced across goroutines. Player
// NPC counts per map are small (script-id band capacity), so there is no
// concurrency benefit worth the synchronization cost.
//
// Deliberately never touches npc/controller.TryClaim/ElectFor/ReleaseFor —
// Player NPC object ids must never enter the controller registry (FR-7.4).
func spawnPlayerNpcsForSession(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, f field.Model) error {
	npcs, err := playernpc.NewProcessor(l, ctx).InMapModelProvider(f)()
	if err != nil {
		return err
	}
	if len(npcs) == 0 {
		return nil
	}
	entries := make([]npcpkt.ImitatedNpc, 0, len(npcs))
	for _, n := range npcs {
		if err := playerNpcAnnounce(l, ctx, wp, npcpkt.NpcSpawnWriter, PlayerNpcSpawn(n).Encode, s); err != nil {
			return err
		}
		entries = append(entries, PlayerNpcImitatedEntry(n))
	}
	return playerNpcAnnounce(l, ctx, wp, npcpkt.NpcImitatedDataWriter, npcpkt.NewImitatedNpcData(entries).Encode, s)
}

package _map

import (
	npcKafka "atlas-channel/kafka/message/npc"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// scriptedNpcAnnounce is the session.Announce seam for scripted-NPC
// SPAWN_NPC packets, extracted as a package-level var (mirrors
// playerNpcAnnounce/doorAnnounce) so tests can stub it without a real
// socket writer.
var scriptedNpcAnnounce = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, writerName string, enc packet.Encode, s session.Model) error {
	return session.Announce(l)(ctx)(wp)(writerName)(enc)(s)
}

// ScriptedNpcSpawn builds the SPAWN_NPC packet for a scripted NPC placed by
// the spawn_npc saga action (task-290 G2/C14).
//
// Cosmic's spawnNpc also sets cy, facing, and rx0/rx1 walk-range bounds on
// the live NPC object (AbstractPlayerInteraction.java:962-973 per
// docs/tasks/task-290-cosmic-map-action-parity/context.md:213), but
// atlas-maps' map/npc.Model does not carry them -- the spawn_npc saga
// payload only threads NpcId/X/Y/Fh through (task-BC brief; C14 never
// recorded the rest). There is no precedent in this codebase that settles
// what they should default to for a saga-spawned NPC (playernpc.Model's
// Cy/RX0/RX1/Dir come from an explicit deploy-time REST input this action
// has no equivalent of), so this substitutes the only values on hand: y
// stands in for cy, x for both rx0 and rx1 (a zero-width walk range -- the
// NPC does not roam), and 0 for facing. These are deliberate placeholders,
// not a verified Cosmic value -- flagged here rather than guessed silently.
func ScriptedNpcSpawn(uniqueId uint32, npcId uint32, x int16, y int16, fh int16) npcpkt.Spawn {
	return npcpkt.NewNpcSpawn(uniqueId, npcId, x, y, 0, uint16(fh), x, x)
}

// handleStatusEventNpcCreated broadcasts a newly placed scripted NPC (task-290
// C14's spawn_npc saga action) to every session already in its field.
// task-BC: without this, spawn_npc registers the NPC server-side in
// atlas-maps but no client ever sees it.
func handleStatusEventNpcCreated(sc server.Model, wp writer.Producer) message.Handler[npcKafka.StatusEvent[npcKafka.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e npcKafka.StatusEvent[npcKafka.CreatedStatusEventBody]) {
		if e.Type != npcKafka.EventStatusTypeCreated {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		spawn := ScriptedNpcSpawn(e.UniqueId, e.Body.NpcId, e.Body.X, e.Body.Y, e.Body.Fh)
		f := sc.Field(e.MapId, e.Instance)
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
			return scriptedNpcAnnounce(l, ctx, wp, npcpkt.NpcSpawnWriter, spawn.Encode, s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn scripted npc [%d] for characters in map [%d].", e.UniqueId, e.MapId)
		}
	}
}

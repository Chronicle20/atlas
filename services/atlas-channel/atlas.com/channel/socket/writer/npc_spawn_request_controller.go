package writer

import (
	"atlas-channel/data/npc"

	npcpkt "github.com/Chronicle20/atlas-packet/npc"
	"github.com/Chronicle20/atlas-socket/packet"
)

const SpawnNPCRequestController = "SpawnNPCRequestController"

func SpawnNPCRequestControllerBody(npc npc.Model, miniMap bool) packet.Encode {
	return npcpkt.NewNpcSpawnRequestController(npc.Id(), npc.Template(), npc.X(), npc.CY(), int32(npc.F()), npc.Fh(), npc.RX0(), npc.RX1(), miniMap).Encode
}

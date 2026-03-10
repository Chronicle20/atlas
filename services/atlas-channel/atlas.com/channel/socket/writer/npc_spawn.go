package writer

import (
	"atlas-channel/data/npc"

	npcpkt "github.com/Chronicle20/atlas-packet/npc"
	"github.com/Chronicle20/atlas-socket/packet"
)

const SpawnNPC = "SpawnNPC"

func SpawnNPCBody(npc npc.Model) packet.Encode {
	return npcpkt.NewNpcSpawn(npc.Id(), npc.Template(), npc.X(), npc.CY(), int32(npc.F()), npc.Fh(), npc.RX0(), npc.RX1()).Encode
}

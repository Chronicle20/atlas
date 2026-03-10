package writer

import (
	"github.com/Chronicle20/atlas-packet/model"
	npcpkt "github.com/Chronicle20/atlas-packet/npc"
	"github.com/Chronicle20/atlas-socket/packet"
)

const NPCAction = "NPCAction"

func NPCActionAnimationBody(objectId uint32, unk byte, unk2 byte) packet.Encode {
	return npcpkt.NewNpcActionAnimation(objectId, unk, unk2).Encode
}

func NPCActionMoveBody(objectId uint32, unk byte, unk2 byte, movePath model.Movement) packet.Encode {
	return npcpkt.NewNpcActionMove(objectId, unk, unk2, movePath).Encode
}

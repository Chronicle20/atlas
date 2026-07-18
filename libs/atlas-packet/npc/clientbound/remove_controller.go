package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// packet-audit:fname CNpcPool::OnNpcChangeController
//
// RemoveController is the flag-0 (remove) arm of OnNpcChangeController: the
// client demotes the NPC to remote control (SetRemoteNpc) and stops running
// its AI/animation locally. Same opcode as SpawnRequestController (the
// flag-1 grant arm); read order: Decode1 flag, Decode4 npc object id, no
// further reads (GMS v95 0x679730, GMS v83 0x6d9a83).
type RemoveController struct {
	id uint32
}

func NewNpcRemoveController(id uint32) RemoveController {
	return RemoveController{id: id}
}

func (m RemoveController) Id() uint32 {
	return m.id
}

func (m RemoveController) Operation() string {
	return NpcSpawnRequestControllerWriter
}

func (m RemoveController) String() string {
	return fmt.Sprintf("id [%d] (remove controller)", m.id)
}

func (m RemoveController) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0)
		w.WriteInt(m.id)
		return w.Bytes()
	}
}

func (m *RemoveController) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadByte() // always 0 (remove arm)
		m.id = r.ReadUint32()
	}
}

package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const NpcRemoveWriter = "RemoveNPC"

// packet-audit:fname CNpcPool::OnNpcLeaveField
//
// Read order confirmed identical across all ten applicable versions: a
// single Decode4 (the NPC's field object id) with no further CInPacket
// reads. Function body is 0x5e bytes in every derived version (GMS v48
// 0x56d5b9, v61 0x5efe4b, v72 0x645f45, v79 0x668ad6, v83 0x6d9a25, v84
// 0x6f0bc8, v87 0x71706a, v92 0x66bd70, v95 0x6792c0, JMS v185 0x720724).
type Remove struct {
	id uint32
}

func NewNpcRemove(id uint32) Remove {
	return Remove{id: id}
}

func (m Remove) Id() uint32 { return m.id }

func (m Remove) Operation() string { return NpcRemoveWriter }

func (m Remove) String() string {
	return fmt.Sprintf("id [%d]", m.id)
}

func (m Remove) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		return w.Bytes()
	}
}

func (m *Remove) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
	}
}

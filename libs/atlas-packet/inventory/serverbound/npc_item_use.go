package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const NpcItemUseHandle = "NpcItemUseHandle"

// NpcItemUse - the remote-NPC item use request, covering the 239xxxx
// (remote-NPC summon) and 545xxxx (remote merchant) families. The item names an
// NPC and opens that NPC's own shop or conversation from anywhere; unlike
// ScriptedItem the item carries no dialogue of its own.
//
// Body is invariant across all nine versions that carry the opcode (v61 through
// jms_v185):
//
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// THERE IS NO LEADING updateTime — contrast ScriptedItem in this same package,
// which has one. Client gate:
// (nItemID/10000 == 545 || nItemID/10000 == 239) && CanSendExclRequest(200, 0),
// note the 200 ms window against ScriptedItem's 500 ms. Two client-side refusal
// arms send nothing at all: field flag bit 18 (0x40000) set, and a
// CUniqueModeless dialog already open.
//
// Absent from gms_v12 and gms_v48.
//
// packet-audit:fname CWvsContext::SendSelectNpcItemUseRequest
type NpcItemUse struct {
	source int16
	itemId uint32
}

func NewNpcItemUse(source int16, itemId uint32) NpcItemUse {
	return NpcItemUse{source: source, itemId: itemId}
}

func (m NpcItemUse) Source() int16  { return m.source }
func (m NpcItemUse) ItemId() uint32 { return m.itemId }

func (m NpcItemUse) Operation() string {
	return NpcItemUseHandle
}

func (m NpcItemUse) String() string {
	return fmt.Sprintf("source [%d], itemId [%d]", m.source, m.itemId)
}

func (m NpcItemUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *NpcItemUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}

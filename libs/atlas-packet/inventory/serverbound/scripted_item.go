package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ScriptedItemHandle = "ScriptedItemHandle"

// ScriptedItem - the 243xxxx scripted-item use request. The item carries its
// own dialogue, keyed by item id and rendered with the avatar named in the
// item's WZ spec/npc node.
//
// Body is invariant across every version that carries the opcode (v72 through
// jms_v185) — a full sweep of all ten IDBs found no divergence, so there is no
// version gating here deliberately:
//
//	Encode4(get_update_time())   // uint32 update time
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// Contrast the sibling NpcItemUse in this package, which has NO leading
// updateTime. Client-side gate: nItemID / 10000 == 243 under
// CanSendExclRequest(500, 0). v83+ additionally calls
// CWvsContext::IsAbleToConsume; v72/v79 do not, so the server performs its own
// ownership and quantity validation on every version.
//
// Absent from gms_v12, gms_v48, gms_v61.
//
// packet-audit:fname CWvsContext::SendScriptRunItemRequest
type ScriptedItem struct {
	updateTime uint32
	source     int16
	itemId     uint32
}

func NewScriptedItem(updateTime uint32, source int16, itemId uint32) ScriptedItem {
	return ScriptedItem{updateTime: updateTime, source: source, itemId: itemId}
}

func (m ScriptedItem) UpdateTime() uint32 { return m.updateTime }
func (m ScriptedItem) Source() int16      { return m.source }
func (m ScriptedItem) ItemId() uint32     { return m.itemId }

func (m ScriptedItem) Operation() string {
	return ScriptedItemHandle
}

func (m ScriptedItem) String() string {
	return fmt.Sprintf("updateTime [%d], source [%d], itemId [%d]", m.updateTime, m.source, m.itemId)
}

func (m ScriptedItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *ScriptedItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}

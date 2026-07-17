package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TeleportRockUseHandle = "TeleportRockUseHandle"

// Use - CWvsContext::SendMapTransferItemUseRequest (USE_TELEPORT_ROCK).
// Layout (design task-124 §1 Q1, version-invariant):
//
//	short nPOS       // USE-inventory slot of the rock
//	int   nItemID    // client-side guard: nItemID/10000 == 232 on this op
//	<RunMapTransferItem target payload — teleportrock.Target>
//	int   updateTime // trailing on all versions (no leading updateTime, even v95)
//
// Valid() is false when the client omitted/truncated the target payload; the
// handler warn-drops such requests (no result packet — the request was
// malformed by the client's own rules).
type Use struct {
	slot       int16
	itemId     uint32
	target     teleportrock.Target
	updateTime uint32
}

func NewUse(slot int16, itemId uint32, target teleportrock.Target, updateTime uint32) Use {
	return Use{slot: slot, itemId: itemId, target: target, updateTime: updateTime}
}

func (m Use) Slot() int16                 { return m.slot }
func (m Use) ItemId() uint32              { return m.itemId }
func (m Use) Target() teleportrock.Target { return m.target }
func (m Use) UpdateTime() uint32          { return m.updateTime }
func (m Use) Valid() bool                 { return m.target.Valid() }
func (m Use) Operation() string           { return TeleportRockUseHandle }

func (m Use) String() string {
	return fmt.Sprintf("Use{slot=%d itemId=%d target=%s updateTime=%d}", m.slot, m.itemId, m.target.String(), m.updateTime)
}

func (m Use) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.slot)
		w.WriteInt(m.itemId)
		m.target.Encode(w)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *Use) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadInt16()
		m.itemId = r.ReadUint32()
		m.target.Decode(l)(r)
		if r.Available() >= 4 {
			m.updateTime = r.ReadUint32()
		}
	}
}

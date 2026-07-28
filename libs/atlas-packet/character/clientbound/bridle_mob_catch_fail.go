package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const BridleMobCatchFailWriter = "BridleMobCatchFail"

// BridleMobCatchFail is the clientbound BRIDLE_MOB_CATCH_FAIL packet
// (CWvsContext::OnBridleMobCatchFail): the server notifies the client that a
// bridle (taming-item) capture attempt failed, so the client shows the right
// chat-log reason message.
//
// Byte layout (IDA-verified, identical across all 5 versions — Decode1 + 2×Decode4):
//   - reason : byte  — the failure-reason selector (0 = too strong, 1 = on
//     cooldown). Decode1, drives the StringPool message branch.
//   - itemId : int32 — the bridle item id, looked up via GetBridleItem (Decode4).
//   - unused : int32 — a trailing Decode4 the client reads but discards.
//
// IDA basis: CWvsContext::OnBridleMobCatchFail — v83 @0xa0800e (`v15 =
// Decode1(a1); v1 = Decode4(a1); Decode4(a1); GetBridleItem(v1)`), v84
// @0xa522fc, v87 @0xa9d692, v95 @0x9d9a80, jms @0xaec5ed — every version reads
// one Decode1 then two Decode4 (the second Decode4's value is never stored).
//
// packet-audit:fname CWvsContext::OnBridleMobCatchFail
type BridleMobCatchFail struct {
	reason byte
	itemId int32
	unused int32
}

func NewBridleMobCatchFail(reason byte, itemId int32, unused int32) BridleMobCatchFail {
	return BridleMobCatchFail{reason: reason, itemId: itemId, unused: unused}
}

func (m BridleMobCatchFail) Reason() byte      { return m.reason }
func (m BridleMobCatchFail) ItemId() int32     { return m.itemId }
func (m BridleMobCatchFail) Unused() int32     { return m.unused }
func (m BridleMobCatchFail) Operation() string { return BridleMobCatchFailWriter }
func (m BridleMobCatchFail) String() string {
	return fmt.Sprintf("reason [%d], itemId [%d], unused [%d]", m.reason, m.itemId, m.unused)
}

func (m BridleMobCatchFail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.reason)
		w.WriteInt32(m.itemId)
		w.WriteInt32(m.unused)
		return w.Bytes()
	}
}

func (m *BridleMobCatchFail) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.reason = r.ReadByte()
		m.itemId = r.ReadInt32()
		m.unused = r.ReadInt32()
	}
}

package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Discrete per-mode "TwoInts-shape" arms of the CITC::OnNormalItemResult
// dispatcher (MTS_OPERATION). Each arm's CITC sub-handler reads exactly two
// Decode4 ints after the dispatcher's Decode1(mode). The downstream use of the
// two ints differs per arm (tab/selectedNo vs notice counts) but the wire read
// order is identical: Decode4 then Decode4. Each mode therefore has its OWN
// discrete struct that FIXES its own mode byte and writes the mode byte THEN the
// two ints — no shared shape struct (task-096: discrete-per-mode rule).
//
// The mode bytes are version-stable across gms_v83 / gms_v84 / gms_v87 /
// gms_v95 (IDA-verified; identical Decode4 then Decode4 read order). jms_v185
// has NO CITC op (registry-absent). Per-mode per-version sub-handler addresses
// are cited above each struct (dispatcher: v83 0x5a4311 / v84 0x5b47c8 /
// v87 0x5d43d0 / v95 0x5771d0).

// MtsResultMoveItcPurchaseItemLtoSDone — the 0x27 MoveITCPurchaseItemLtoSDone
// arm. The CITC sub-handler reads Decode4(tab) -> CCtrlTab::SetTab(tab-1) and
// Decode4(selectedNo). The trailing m_bITCRequestSent=0 store is a member write,
// not a wire read.
// Sub-handler addresses: v83 0x5a4d68 / v84 0x5b5258 / v87 0x5d4e58 / v95 0x5760a0.
//
// packet-audit:fname CITC::OnNormalItemResult#MoveItcPurchaseItemLtoSDone
type MtsResultMoveItcPurchaseItemLtoSDone struct {
	mode       byte
	tab        uint32
	selectedNo uint32
}

func NewMtsResultMoveItcPurchaseItemLtoSDone(tab uint32, selectedNo uint32) MtsResultMoveItcPurchaseItemLtoSDone {
	return MtsResultMoveItcPurchaseItemLtoSDone{mode: 0x27, tab: tab, selectedNo: selectedNo}
}

func (m MtsResultMoveItcPurchaseItemLtoSDone) Mode() byte         { return m.mode }
func (m MtsResultMoveItcPurchaseItemLtoSDone) Tab() uint32        { return m.tab }
func (m MtsResultMoveItcPurchaseItemLtoSDone) SelectedNo() uint32 { return m.selectedNo }
func (m MtsResultMoveItcPurchaseItemLtoSDone) Operation() string  { return MtsOperationWriter }
func (m MtsResultMoveItcPurchaseItemLtoSDone) String() string {
	return fmt.Sprintf("mts move itc purchase item ltos done mode [%d] tab [%d] selectedNo [%d]", m.mode, m.tab, m.selectedNo)
}

func (m MtsResultMoveItcPurchaseItemLtoSDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)      // dispatcher mode byte (0x27)
		w.WriteInt(m.tab)        // Decode4 tab (-> CCtrlTab::SetTab(tab-1))
		w.WriteInt(m.selectedNo) // Decode4 selectedNo
		return w.Bytes()
	}
}

func (m *MtsResultMoveItcPurchaseItemLtoSDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.tab = r.ReadUint32()
		m.selectedNo = r.ReadUint32()
	}
}

// MtsResultNotifyCancelWishResult — the 0x3D NotifyCancelWishResult arm. The
// CITC sub-handler reads Decode4(countA) and Decode4(countB); each >0 guards a
// StringPool notice. The wire read order is Decode4 then Decode4.
// Sub-handler addresses: v83 0x5a523e / v84 0x5b56f5 / v87 0x5d52f8 / v95 0x576f00.
//
// packet-audit:fname CITC::OnNormalItemResult#NotifyCancelWishResult
type MtsResultNotifyCancelWishResult struct {
	mode   byte
	countA uint32
	countB uint32
}

func NewMtsResultNotifyCancelWishResult(countA uint32, countB uint32) MtsResultNotifyCancelWishResult {
	return MtsResultNotifyCancelWishResult{mode: 0x3D, countA: countA, countB: countB}
}

func (m MtsResultNotifyCancelWishResult) Mode() byte        { return m.mode }
func (m MtsResultNotifyCancelWishResult) CountA() uint32    { return m.countA }
func (m MtsResultNotifyCancelWishResult) CountB() uint32    { return m.countB }
func (m MtsResultNotifyCancelWishResult) Operation() string { return MtsOperationWriter }
func (m MtsResultNotifyCancelWishResult) String() string {
	return fmt.Sprintf("mts notify cancel wish result mode [%d] countA [%d] countB [%d]", m.mode, m.countA, m.countB)
}

func (m MtsResultNotifyCancelWishResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)  // dispatcher mode byte (0x3D)
		w.WriteInt(m.countA) // Decode4 first notice count
		w.WriteInt(m.countB) // Decode4 second notice count
		return w.Bytes()
	}
}

func (m *MtsResultNotifyCancelWishResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.countA = r.ReadUint32()
		m.countB = r.ReadUint32()
	}
}

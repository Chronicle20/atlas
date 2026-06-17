package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Discrete per-mode "Reason-shape" arms of the CITC::OnNormalItemResult
// dispatcher (MTS_OPERATION). Each arm's CITC sub-handler reads a single
// Decode1 fail-reason byte after the dispatcher's Decode1(mode) — the reason is
// routed to NoticeFailReason (or a reason-keyed StringPool::GetString). No
// further CInPacket::Decode* is performed (the reason==73/65 transfer-field
// re-send branches read NO additional wire bytes). Each mode therefore has its
// OWN discrete struct that FIXES its own mode byte and writes the mode byte THEN
// the reason byte — no shared shape struct (task-096: discrete-per-mode rule).
//
// The mode bytes are version-stable across gms_v83 / gms_v84 / gms_v87 /
// gms_v95 (IDA-verified; the case labels are identical and the sub-handler
// bodies are byte-identical in shape). jms_v185 has NO CITC op (registry-absent).
// Per-mode per-version sub-handler addresses are cited in the doc comment above
// each struct (dispatcher: v83 0x5a4311 / v84 0x5b47c8 / v87 0x5d43d0 /
// v95 0x5771d0).

// MtsResultGetItcListFailed — the 0x16 GetITCListFailed arm. The CITC
// sub-handler reads Decode1(reason) -> NoticeFailReason (reason==73 also
// re-sends the transfer-field packet, which reads no further bytes). The wire
// after the dispatcher mode byte is exactly one Decode1 reason byte.
// Sub-handler addresses: v83 0x5a4882 / v84 0x5b4d72 / v87 0x5d4972 / v95 0x575f70.
//
// packet-audit:fname CITC::OnNormalItemResult#GetItcListFailed
type MtsResultGetItcListFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultGetItcListFailed(reason byte) MtsResultGetItcListFailed {
	return MtsResultGetItcListFailed{mode: 0x16, reason: reason}
}

func (m MtsResultGetItcListFailed) Mode() byte        { return m.mode }
func (m MtsResultGetItcListFailed) Reason() byte      { return m.reason }
func (m MtsResultGetItcListFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultGetItcListFailed) String() string {
	return fmt.Sprintf("mts get itc list failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultGetItcListFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x16)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultGetItcListFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultGetSearchItcListFailed — the 0x18 GetSearchITCListFailed arm. The
// CITC sub-handler reads Decode1(reason) -> NoticeFailReason (reason==73 also
// re-sends the transfer-field packet, which reads no further bytes).
// Sub-handler addresses: v83 0x5a49e3 / v84 0x5b4ed3 / v87 0x5d4ad3 / v95 0x575fa0.
//
// packet-audit:fname CITC::OnNormalItemResult#GetSearchItcListFailed
type MtsResultGetSearchItcListFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultGetSearchItcListFailed(reason byte) MtsResultGetSearchItcListFailed {
	return MtsResultGetSearchItcListFailed{mode: 0x18, reason: reason}
}

func (m MtsResultGetSearchItcListFailed) Mode() byte        { return m.mode }
func (m MtsResultGetSearchItcListFailed) Reason() byte      { return m.reason }
func (m MtsResultGetSearchItcListFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultGetSearchItcListFailed) String() string {
	return fmt.Sprintf("mts get search itc list failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultGetSearchItcListFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x18)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultGetSearchItcListFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultSaleCurrentItemToWishFailed — the 0x20 SaleCurrentItemToWishFailed
// arm. The CITC sub-handler reads Decode1(reason) -> reason-keyed
// StringPool::GetString notice (80/82|83/else). The wire after the dispatcher
// mode byte is exactly one Decode1 reason byte.
// Sub-handler addresses: v83 0x5a46f0 / v84 0x5b4be0 / v87 0x5d47c4 / v95 0x575d70.
//
// packet-audit:fname CITC::OnNormalItemResult#SaleCurrentItemToWishFailed
type MtsResultSaleCurrentItemToWishFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultSaleCurrentItemToWishFailed(reason byte) MtsResultSaleCurrentItemToWishFailed {
	return MtsResultSaleCurrentItemToWishFailed{mode: 0x20, reason: reason}
}

func (m MtsResultSaleCurrentItemToWishFailed) Mode() byte        { return m.mode }
func (m MtsResultSaleCurrentItemToWishFailed) Reason() byte      { return m.reason }
func (m MtsResultSaleCurrentItemToWishFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultSaleCurrentItemToWishFailed) String() string {
	return fmt.Sprintf("mts sale current item to wish failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultSaleCurrentItemToWishFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x20)
		w.WriteByte(m.reason) // Decode1 fail reason -> reason-keyed StringPool notice
		return w.Bytes()
	}
}

func (m *MtsResultSaleCurrentItemToWishFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultGetUserPurchaseItemFailed — the 0x22 GetUserPurchaseItemFailed arm.
// The CITC sub-handler reads Decode1(reason) -> NoticeFailReason (reason==73
// also re-sends the transfer-field packet, which reads no further bytes).
// Sub-handler addresses: v83 0x5a4c2a / v84 0x5b511a / v87 0x5d4d1a / v95 0x575fd0.
//
// packet-audit:fname CITC::OnNormalItemResult#GetUserPurchaseItemFailed
type MtsResultGetUserPurchaseItemFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultGetUserPurchaseItemFailed(reason byte) MtsResultGetUserPurchaseItemFailed {
	return MtsResultGetUserPurchaseItemFailed{mode: 0x22, reason: reason}
}

func (m MtsResultGetUserPurchaseItemFailed) Mode() byte        { return m.mode }
func (m MtsResultGetUserPurchaseItemFailed) Reason() byte      { return m.reason }
func (m MtsResultGetUserPurchaseItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultGetUserPurchaseItemFailed) String() string {
	return fmt.Sprintf("mts get user purchase item failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultGetUserPurchaseItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x22)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultGetUserPurchaseItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultGetUserSaleItemFailed — the 0x24 GetUserSaleItemFailed arm. The CITC
// sub-handler reads Decode1(reason) -> NoticeFailReason (reason==73 also
// re-sends the transfer-field packet, which reads no further bytes).
// Sub-handler addresses: v83 0x5a4ce7 / v84 0x5b51d7 / v87 0x5d4dd7 / v95 0x576000.
//
// packet-audit:fname CITC::OnNormalItemResult#GetUserSaleItemFailed
type MtsResultGetUserSaleItemFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultGetUserSaleItemFailed(reason byte) MtsResultGetUserSaleItemFailed {
	return MtsResultGetUserSaleItemFailed{mode: 0x24, reason: reason}
}

func (m MtsResultGetUserSaleItemFailed) Mode() byte        { return m.mode }
func (m MtsResultGetUserSaleItemFailed) Reason() byte      { return m.reason }
func (m MtsResultGetUserSaleItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultGetUserSaleItemFailed) String() string {
	return fmt.Sprintf("mts get user sale item failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultGetUserSaleItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x24)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultGetUserSaleItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultCancelSaleItemFailed — the 0x26 CancelSaleItemFailed arm. The CITC
// sub-handler reads exactly Decode1(reason) -> NoticeFailReason with NO further
// CInPacket::Decode*.
// Sub-handler addresses: v83 0x5a4d49 / v84 0x5b5239 / v87 0x5d4e39 / v95 0x576070.
//
// packet-audit:fname CITC::OnNormalItemResult#CancelSaleItemFailed
type MtsResultCancelSaleItemFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultCancelSaleItemFailed(reason byte) MtsResultCancelSaleItemFailed {
	return MtsResultCancelSaleItemFailed{mode: 0x26, reason: reason}
}

func (m MtsResultCancelSaleItemFailed) Mode() byte        { return m.mode }
func (m MtsResultCancelSaleItemFailed) Reason() byte      { return m.reason }
func (m MtsResultCancelSaleItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultCancelSaleItemFailed) String() string {
	return fmt.Sprintf("mts cancel sale item failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultCancelSaleItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x26)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultCancelSaleItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultMoveItcPurchaseItemLtoSFailed — the 0x28 MoveITCPurchaseItemLtoSFailed
// arm. The CITC sub-handler reads Decode1(reason) -> NoticeFailReason
// (reason==65 re-sends the transfer-field packet, which reads no further bytes).
// Sub-handler addresses: v83 0x5a4dcf / v84 0x5b52bf / v87 0x5d4ec2 / v95 0x576110.
//
// packet-audit:fname CITC::OnNormalItemResult#MoveItcPurchaseItemLtoSFailed
type MtsResultMoveItcPurchaseItemLtoSFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultMoveItcPurchaseItemLtoSFailed(reason byte) MtsResultMoveItcPurchaseItemLtoSFailed {
	return MtsResultMoveItcPurchaseItemLtoSFailed{mode: 0x28, reason: reason}
}

func (m MtsResultMoveItcPurchaseItemLtoSFailed) Mode() byte        { return m.mode }
func (m MtsResultMoveItcPurchaseItemLtoSFailed) Reason() byte      { return m.reason }
func (m MtsResultMoveItcPurchaseItemLtoSFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultMoveItcPurchaseItemLtoSFailed) String() string {
	return fmt.Sprintf("mts move itc purchase item ltos failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultMoveItcPurchaseItemLtoSFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x28)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultMoveItcPurchaseItemLtoSFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

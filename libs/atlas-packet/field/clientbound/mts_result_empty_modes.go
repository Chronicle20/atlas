package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Discrete per-mode notice-only ("Empty-shape") arms of the
// CITC::OnNormalItemResult dispatcher (MTS_OPERATION). Each arm's CITC
// sub-handler reads NOTHING from the packet after the dispatcher's Decode1(mode)
// — it shows a StringPool::GetString + CUtilDlg::Notice and clears
// m_bITCRequestSent (a member store, not a wire read). Each mode therefore has
// its OWN discrete struct that FIXES its own mode byte and writes exactly that
// one byte — no shared shape struct (task-096: discrete-per-mode rule).
//
// The mode bytes are version-stable across gms_v83 / gms_v84 / gms_v87 /
// gms_v95 (IDA-verified; the case labels 0x15..0x3E are identical and the
// sub-handler bodies are byte-identical in shape). jms_v185 has NO CITC op
// (registry-absent). Per-mode per-version sub-handler addresses are cited in the
// doc comment above each struct (dispatcher: v83 0x5a4311 / v84 0x5b47c8 /
// v87 0x5d43d0 / v95 0x5771d0).

// MtsResultRegisterSaleEntryDone — the 0x1D RegisterSaleEntryDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4674 / v84 0x5b4b64 / v87 0x5d4748 / v95 0x575cd0.
//
// packet-audit:fname CITC::OnNormalItemResult#RegisterSaleEntryDone
type MtsResultRegisterSaleEntryDone struct {
	mode byte
}

func NewMtsResultRegisterSaleEntryDone() MtsResultRegisterSaleEntryDone {
	return MtsResultRegisterSaleEntryDone{mode: 0x1D}
}

func (m MtsResultRegisterSaleEntryDone) Mode() byte        { return m.mode }
func (m MtsResultRegisterSaleEntryDone) Operation() string { return MtsOperationWriter }
func (m MtsResultRegisterSaleEntryDone) String() string {
	return fmt.Sprintf("mts register sale entry done mode [%d]", m.mode)
}

func (m MtsResultRegisterSaleEntryDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x1D); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultRegisterSaleEntryDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultSaleCurrentItemToWishDone — the 0x1F SaleCurrentItemToWishDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a46b2 / v84 0x5b4ba2 / v87 0x5d4786 / v95 0x575d20.
//
// packet-audit:fname CITC::OnNormalItemResult#SaleCurrentItemToWishDone
type MtsResultSaleCurrentItemToWishDone struct {
	mode byte
}

func NewMtsResultSaleCurrentItemToWishDone() MtsResultSaleCurrentItemToWishDone {
	return MtsResultSaleCurrentItemToWishDone{mode: 0x1F}
}

func (m MtsResultSaleCurrentItemToWishDone) Mode() byte        { return m.mode }
func (m MtsResultSaleCurrentItemToWishDone) Operation() string { return MtsOperationWriter }
func (m MtsResultSaleCurrentItemToWishDone) String() string {
	return fmt.Sprintf("mts sale current item to wish done mode [%d]", m.mode)
}

func (m MtsResultSaleCurrentItemToWishDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x1F); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultSaleCurrentItemToWishDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultCancelSaleItemDone — the 0x25 CancelSaleItemDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4d14 / v84 0x5b5204 / v87 0x5d4e04 / v95 0x576030.
//
// packet-audit:fname CITC::OnNormalItemResult#CancelSaleItemDone
type MtsResultCancelSaleItemDone struct {
	mode byte
}

func NewMtsResultCancelSaleItemDone() MtsResultCancelSaleItemDone {
	return MtsResultCancelSaleItemDone{mode: 0x25}
}

func (m MtsResultCancelSaleItemDone) Mode() byte        { return m.mode }
func (m MtsResultCancelSaleItemDone) Operation() string { return MtsOperationWriter }
func (m MtsResultCancelSaleItemDone) String() string {
	return fmt.Sprintf("mts cancel sale item done mode [%d]", m.mode)
}

func (m MtsResultCancelSaleItemDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x25); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultCancelSaleItemDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultSetZzimDone — the 0x29 SetZzimDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4dfc / v84 0x5b52ec / v87 0x5d4eef / v95 0x576140.
//
// packet-audit:fname CITC::OnNormalItemResult#SetZzimDone
type MtsResultSetZzimDone struct {
	mode byte
}

func NewMtsResultSetZzimDone() MtsResultSetZzimDone {
	return MtsResultSetZzimDone{mode: 0x29}
}

func (m MtsResultSetZzimDone) Mode() byte        { return m.mode }
func (m MtsResultSetZzimDone) Operation() string { return MtsOperationWriter }
func (m MtsResultSetZzimDone) String() string {
	return fmt.Sprintf("mts set zzim done mode [%d]", m.mode)
}

func (m MtsResultSetZzimDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x29); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultSetZzimDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultSetZzimFailed — the 0x2A SetZzimFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4e31 / v84 0x5b5321 / v87 0x5d4f24 / v95 0x576180.
//
// packet-audit:fname CITC::OnNormalItemResult#SetZzimFailed
type MtsResultSetZzimFailed struct {
	mode byte
}

func NewMtsResultSetZzimFailed() MtsResultSetZzimFailed {
	return MtsResultSetZzimFailed{mode: 0x2A}
}

func (m MtsResultSetZzimFailed) Mode() byte        { return m.mode }
func (m MtsResultSetZzimFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultSetZzimFailed) String() string {
	return fmt.Sprintf("mts set zzim failed mode [%d]", m.mode)
}

func (m MtsResultSetZzimFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2A); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultSetZzimFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultDeleteZzimDone — the 0x2B DeleteZzimDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4e66 / v84 0x5b5356 / v87 0x5d4f59 / v95 0x5761c0.
//
// packet-audit:fname CITC::OnNormalItemResult#DeleteZzimDone
type MtsResultDeleteZzimDone struct {
	mode byte
}

func NewMtsResultDeleteZzimDone() MtsResultDeleteZzimDone {
	return MtsResultDeleteZzimDone{mode: 0x2B}
}

func (m MtsResultDeleteZzimDone) Mode() byte        { return m.mode }
func (m MtsResultDeleteZzimDone) Operation() string { return MtsOperationWriter }
func (m MtsResultDeleteZzimDone) String() string {
	return fmt.Sprintf("mts delete zzim done mode [%d]", m.mode)
}

func (m MtsResultDeleteZzimDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2B); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultDeleteZzimDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultDeleteZzimFailed — the 0x2C DeleteZzimFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4e91 / v84 0x5b5381 / v87 0x5d4f84 / v95 0x5761f0.
//
// packet-audit:fname CITC::OnNormalItemResult#DeleteZzimFailed
type MtsResultDeleteZzimFailed struct {
	mode byte
}

func NewMtsResultDeleteZzimFailed() MtsResultDeleteZzimFailed {
	return MtsResultDeleteZzimFailed{mode: 0x2C}
}

func (m MtsResultDeleteZzimFailed) Mode() byte        { return m.mode }
func (m MtsResultDeleteZzimFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultDeleteZzimFailed) String() string {
	return fmt.Sprintf("mts delete zzim failed mode [%d]", m.mode)
}

func (m MtsResultDeleteZzimFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2C); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultDeleteZzimFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultLoadWishSaleListFailed — the 0x2E LoadWishSaleListFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4fdc / v84 0x5b54cc / v87 0x5d50cf / v95 0x576230.
//
// packet-audit:fname CITC::OnNormalItemResult#LoadWishSaleListFailed
type MtsResultLoadWishSaleListFailed struct {
	mode byte
}

func NewMtsResultLoadWishSaleListFailed() MtsResultLoadWishSaleListFailed {
	return MtsResultLoadWishSaleListFailed{mode: 0x2E}
}

func (m MtsResultLoadWishSaleListFailed) Mode() byte        { return m.mode }
func (m MtsResultLoadWishSaleListFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultLoadWishSaleListFailed) String() string {
	return fmt.Sprintf("mts load wish sale list failed mode [%d]", m.mode)
}

func (m MtsResultLoadWishSaleListFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2E); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultLoadWishSaleListFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyWishDone — the 0x2F BuyWishDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5011 / v84 0x5b5501 / v87 0x5d5104 / v95 0x576270.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyWishDone
type MtsResultBuyWishDone struct {
	mode byte
}

func NewMtsResultBuyWishDone() MtsResultBuyWishDone {
	return MtsResultBuyWishDone{mode: 0x2F}
}

func (m MtsResultBuyWishDone) Mode() byte        { return m.mode }
func (m MtsResultBuyWishDone) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyWishDone) String() string {
	return fmt.Sprintf("mts buy wish done mode [%d]", m.mode)
}

func (m MtsResultBuyWishDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2F); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyWishDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyWishFailed — the 0x30 BuyWishFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a503c / v84 0x5b552c / v87 0x5d512f / v95 0x5762a0.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyWishFailed
type MtsResultBuyWishFailed struct {
	mode byte
}

func NewMtsResultBuyWishFailed() MtsResultBuyWishFailed {
	return MtsResultBuyWishFailed{mode: 0x30}
}

func (m MtsResultBuyWishFailed) Mode() byte        { return m.mode }
func (m MtsResultBuyWishFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyWishFailed) String() string {
	return fmt.Sprintf("mts buy wish failed mode [%d]", m.mode)
}

func (m MtsResultBuyWishFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x30); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyWishFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultCancelWishDone — the 0x31 CancelWishDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5071 / v84 0x5b5561 / v87 0x5d5164 / v95 0x5762e0.
//
// packet-audit:fname CITC::OnNormalItemResult#CancelWishDone
type MtsResultCancelWishDone struct {
	mode byte
}

func NewMtsResultCancelWishDone() MtsResultCancelWishDone {
	return MtsResultCancelWishDone{mode: 0x31}
}

func (m MtsResultCancelWishDone) Mode() byte        { return m.mode }
func (m MtsResultCancelWishDone) Operation() string { return MtsOperationWriter }
func (m MtsResultCancelWishDone) String() string {
	return fmt.Sprintf("mts cancel wish done mode [%d]", m.mode)
}

func (m MtsResultCancelWishDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x31); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultCancelWishDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultCancelWishFailed — the 0x32 CancelWishFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a50df / v84 0x5b5596 / v87 0x5d5199 / v95 0x576320.
//
// packet-audit:fname CITC::OnNormalItemResult#CancelWishFailed
type MtsResultCancelWishFailed struct {
	mode byte
}

func NewMtsResultCancelWishFailed() MtsResultCancelWishFailed {
	return MtsResultCancelWishFailed{mode: 0x32}
}

func (m MtsResultCancelWishFailed) Mode() byte        { return m.mode }
func (m MtsResultCancelWishFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultCancelWishFailed) String() string {
	return fmt.Sprintf("mts cancel wish failed mode [%d]", m.mode)
}

func (m MtsResultCancelWishFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x32); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultCancelWishFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyItemDone — the 0x33 BuyItemDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5114 / v84 0x5b55cb / v87 0x5d51ce / v95 0x576360.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyItemDone
type MtsResultBuyItemDone struct {
	mode byte
}

func NewMtsResultBuyItemDone() MtsResultBuyItemDone {
	return MtsResultBuyItemDone{mode: 0x33}
}

func (m MtsResultBuyItemDone) Mode() byte        { return m.mode }
func (m MtsResultBuyItemDone) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyItemDone) String() string {
	return fmt.Sprintf("mts buy item done mode [%d]", m.mode)
}

func (m MtsResultBuyItemDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x33); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyItemDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyItemFailed — the 0x34 BuyItemFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a513f / v84 0x5b55f6 / v87 0x5d51f9 / v95 0x576390.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyItemFailed
type MtsResultBuyItemFailed struct {
	mode byte
}

func NewMtsResultBuyItemFailed() MtsResultBuyItemFailed {
	return MtsResultBuyItemFailed{mode: 0x34}
}

func (m MtsResultBuyItemFailed) Mode() byte        { return m.mode }
func (m MtsResultBuyItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyItemFailed) String() string {
	return fmt.Sprintf("mts buy item failed mode [%d]", m.mode)
}

func (m MtsResultBuyItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x34); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyZzimItemDone — the 0x35 BuyZzimItemDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5174 / v84 0x5b562b / v87 0x5d522e / v95 0x5763d0.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyZzimItemDone
type MtsResultBuyZzimItemDone struct {
	mode byte
}

func NewMtsResultBuyZzimItemDone() MtsResultBuyZzimItemDone {
	return MtsResultBuyZzimItemDone{mode: 0x35}
}

func (m MtsResultBuyZzimItemDone) Mode() byte        { return m.mode }
func (m MtsResultBuyZzimItemDone) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyZzimItemDone) String() string {
	return fmt.Sprintf("mts buy zzim item done mode [%d]", m.mode)
}

func (m MtsResultBuyZzimItemDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x35); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyZzimItemDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyZzimItemFailed — the 0x36 BuyZzimItemFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a519f / v84 0x5b5656 / v87 0x5d5259 / v95 0x576400.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyZzimItemFailed
type MtsResultBuyZzimItemFailed struct {
	mode byte
}

func NewMtsResultBuyZzimItemFailed() MtsResultBuyZzimItemFailed {
	return MtsResultBuyZzimItemFailed{mode: 0x36}
}

func (m MtsResultBuyZzimItemFailed) Mode() byte        { return m.mode }
func (m MtsResultBuyZzimItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyZzimItemFailed) String() string {
	return fmt.Sprintf("mts buy zzim item failed mode [%d]", m.mode)
}

func (m MtsResultBuyZzimItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x36); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyZzimItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultRegisterWishItemDone — the 0x37 RegisterWishItemDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a51d4 / v84 0x5b568b / v87 0x5d528e / v95 0x576440.
//
// packet-audit:fname CITC::OnNormalItemResult#RegisterWishItemDone
type MtsResultRegisterWishItemDone struct {
	mode byte
}

func NewMtsResultRegisterWishItemDone() MtsResultRegisterWishItemDone {
	return MtsResultRegisterWishItemDone{mode: 0x37}
}

func (m MtsResultRegisterWishItemDone) Mode() byte        { return m.mode }
func (m MtsResultRegisterWishItemDone) Operation() string { return MtsOperationWriter }
func (m MtsResultRegisterWishItemDone) String() string {
	return fmt.Sprintf("mts register wish item done mode [%d]", m.mode)
}

func (m MtsResultRegisterWishItemDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x37); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultRegisterWishItemDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultRegisterWishItemFailed — the 0x38 RegisterWishItemFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5209 / v84 0x5b56c0 / v87 0x5d52c3 / v95 0x576480.
//
// packet-audit:fname CITC::OnNormalItemResult#RegisterWishItemFailed
type MtsResultRegisterWishItemFailed struct {
	mode byte
}

func NewMtsResultRegisterWishItemFailed() MtsResultRegisterWishItemFailed {
	return MtsResultRegisterWishItemFailed{mode: 0x38}
}

func (m MtsResultRegisterWishItemFailed) Mode() byte        { return m.mode }
func (m MtsResultRegisterWishItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultRegisterWishItemFailed) String() string {
	return fmt.Sprintf("mts register wish item failed mode [%d]", m.mode)
}

func (m MtsResultRegisterWishItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x38); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultRegisterWishItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBidAuctionFailed — the 0x3C BidAuctionFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5444 / v84 0x5b58fb / v87 0x5d54fe / v95 0x5764c0.
//
// packet-audit:fname CITC::OnNormalItemResult#BidAuctionFailed
type MtsResultBidAuctionFailed struct {
	mode byte
}

func NewMtsResultBidAuctionFailed() MtsResultBidAuctionFailed {
	return MtsResultBidAuctionFailed{mode: 0x3C}
}

func (m MtsResultBidAuctionFailed) Mode() byte        { return m.mode }
func (m MtsResultBidAuctionFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultBidAuctionFailed) String() string {
	return fmt.Sprintf("mts bid auction failed mode [%d]", m.mode)
}

func (m MtsResultBidAuctionFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x3C); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBidAuctionFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

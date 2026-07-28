package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const NPCShopOperationWriter = "NPCShopOperation"

// The CONFIRM_SHOP_TRANSACTION (CShopDlg::OnPacket) mode-only "notice" arms each
// get a DISCRETE struct: one struct per dispatcher mode, no shared shape. Each
// arm reads exactly one byte (the mode discriminator) then shows a StringPool
// Notice — identical wire shape, distinct mode key. Mode bytes trace to
// docs/packets/dispatchers/npc_shop_operation.yaml (IDA-verified per version).
// Each struct FIXES its own operation key via its body func in
// shop_operation_body.go (never accepts the key from the caller); the discrete
// constructor receives only the version-resolved mode byte.

// ShopOperationOk - CONFIRM_SHOP_TRANSACTION OK arm (mode-only notice).
// packet-audit:fname CShopDlg::OnPacket#Ok
type ShopOperationOk struct {
	mode byte
}

func NewShopOperationOk(mode byte) ShopOperationOk { return ShopOperationOk{mode: mode} }

func (m ShopOperationOk) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationOk) String() string    { return fmt.Sprintf("shop operation OK mode [%d]", m.mode) }

func (m ShopOperationOk) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationOk) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopOperationOutOfStock - CONFIRM_SHOP_TRANSACTION OUT_OF_STOCK arm.
// packet-audit:fname CShopDlg::OnPacket#OutOfStock
type ShopOperationOutOfStock struct {
	mode byte
}

func NewShopOperationOutOfStock(mode byte) ShopOperationOutOfStock {
	return ShopOperationOutOfStock{mode: mode}
}

func (m ShopOperationOutOfStock) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationOutOfStock) String() string {
	return fmt.Sprintf("shop operation out of stock mode [%d]", m.mode)
}

func (m ShopOperationOutOfStock) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationOutOfStock) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopOperationNotEnoughMoney - CONFIRM_SHOP_TRANSACTION NOT_ENOUGH_MONEY arm.
// packet-audit:fname CShopDlg::OnPacket#NotEnoughMoney
type ShopOperationNotEnoughMoney struct {
	mode byte
}

func NewShopOperationNotEnoughMoney(mode byte) ShopOperationNotEnoughMoney {
	return ShopOperationNotEnoughMoney{mode: mode}
}

func (m ShopOperationNotEnoughMoney) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationNotEnoughMoney) String() string {
	return fmt.Sprintf("shop operation not enough money mode [%d]", m.mode)
}

func (m ShopOperationNotEnoughMoney) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationNotEnoughMoney) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopOperationInventoryFull - CONFIRM_SHOP_TRANSACTION INVENTORY_FULL arm.
// packet-audit:fname CShopDlg::OnPacket#InventoryFull
type ShopOperationInventoryFull struct {
	mode byte
}

func NewShopOperationInventoryFull(mode byte) ShopOperationInventoryFull {
	return ShopOperationInventoryFull{mode: mode}
}

func (m ShopOperationInventoryFull) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationInventoryFull) String() string {
	return fmt.Sprintf("shop operation inventory full mode [%d]", m.mode)
}

func (m ShopOperationInventoryFull) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationInventoryFull) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopOperationOutOfStock2 - CONFIRM_SHOP_TRANSACTION OUT_OF_STOCK_2 arm.
// packet-audit:fname CShopDlg::OnPacket#OutOfStock2
type ShopOperationOutOfStock2 struct {
	mode byte
}

func NewShopOperationOutOfStock2(mode byte) ShopOperationOutOfStock2 {
	return ShopOperationOutOfStock2{mode: mode}
}

func (m ShopOperationOutOfStock2) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationOutOfStock2) String() string {
	return fmt.Sprintf("shop operation out of stock 2 mode [%d]", m.mode)
}

func (m ShopOperationOutOfStock2) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationOutOfStock2) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopOperationOutOfStock3 - CONFIRM_SHOP_TRANSACTION OUT_OF_STOCK_3 arm.
// packet-audit:fname CShopDlg::OnPacket#OutOfStock3
type ShopOperationOutOfStock3 struct {
	mode byte
}

func NewShopOperationOutOfStock3(mode byte) ShopOperationOutOfStock3 {
	return ShopOperationOutOfStock3{mode: mode}
}

func (m ShopOperationOutOfStock3) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationOutOfStock3) String() string {
	return fmt.Sprintf("shop operation out of stock 3 mode [%d]", m.mode)
}

func (m ShopOperationOutOfStock3) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationOutOfStock3) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopOperationNotEnoughMoney2 - CONFIRM_SHOP_TRANSACTION NOT_ENOUGH_MONEY_2 arm.
// packet-audit:fname CShopDlg::OnPacket#NotEnoughMoney2
type ShopOperationNotEnoughMoney2 struct {
	mode byte
}

func NewShopOperationNotEnoughMoney2(mode byte) ShopOperationNotEnoughMoney2 {
	return ShopOperationNotEnoughMoney2{mode: mode}
}

func (m ShopOperationNotEnoughMoney2) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationNotEnoughMoney2) String() string {
	return fmt.Sprintf("shop operation not enough money 2 mode [%d]", m.mode)
}

func (m ShopOperationNotEnoughMoney2) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationNotEnoughMoney2) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopOperationNeedMoreItems - CONFIRM_SHOP_TRANSACTION NEED_MORE_ITEMS arm.
// packet-audit:fname CShopDlg::OnPacket#NeedMoreItems
type ShopOperationNeedMoreItems struct {
	mode byte
}

func NewShopOperationNeedMoreItems(mode byte) ShopOperationNeedMoreItems {
	return ShopOperationNeedMoreItems{mode: mode}
}

func (m ShopOperationNeedMoreItems) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationNeedMoreItems) String() string {
	return fmt.Sprintf("shop operation need more items mode [%d]", m.mode)
}

func (m ShopOperationNeedMoreItems) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationNeedMoreItems) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopOperationTradeLimit - CONFIRM_SHOP_TRANSACTION TRADE_LIMIT arm.
// packet-audit:fname CShopDlg::OnPacket#TradeLimit
type ShopOperationTradeLimit struct {
	mode byte
}

func NewShopOperationTradeLimit(mode byte) ShopOperationTradeLimit {
	return ShopOperationTradeLimit{mode: mode}
}

func (m ShopOperationTradeLimit) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationTradeLimit) String() string {
	return fmt.Sprintf("shop operation trade limit mode [%d]", m.mode)
}

func (m ShopOperationTradeLimit) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationTradeLimit) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// CONFIRM_SHOP_TRANSACTION generic-error arms. The dispatcher reads Decode1
// mode + Decode1 hasReason, then conditionally DecodeStr reason. Atlas splits
// this into two DISCRETE structs, one per operation key (no shared shape):
//   - ShopOperationGenericError       — hasReason=false, no string (GENERIC_ERROR)
//   - ShopOperationGenericErrorWithReason — hasReason=true + string (GENERIC_ERROR_WITH_REASON)
// Mode bytes per docs/packets/dispatchers/npc_shop_operation.yaml (IDA-verified):
// GENERIC_ERROR=17 (all versions); GENERIC_ERROR_WITH_REASON=17 in gms_v83/v84/
// v87, 19 in gms_v95, and VERSION-ABSENT in jms_v185 (jms case 0x13 has no
// Decode1+DecodeStr arm). Each struct's body func resolves its own fixed key.

// ShopOperationGenericError - CONFIRM_SHOP_TRANSACTION GENERIC_ERROR arm (no reason).
// packet-audit:fname CShopDlg::OnPacket#GenericError
type ShopOperationGenericError struct {
	mode byte
}

func NewShopOperationGenericError(mode byte) ShopOperationGenericError {
	return ShopOperationGenericError{mode: mode}
}

func (m ShopOperationGenericError) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationGenericError) String() string {
	return fmt.Sprintf("shop operation generic error mode [%d]", m.mode)
}

func (m ShopOperationGenericError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(false)
		return w.Bytes()
	}
}

func (m *ShopOperationGenericError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadBool()
	}
}

// ShopOperationGenericErrorWithReason - CONFIRM_SHOP_TRANSACTION GENERIC_ERROR_WITH_REASON arm.
// packet-audit:fname CShopDlg::OnPacket#GenericErrorWithReason
type ShopOperationGenericErrorWithReason struct {
	mode   byte
	reason string
}

func NewShopOperationGenericErrorWithReason(mode byte, reason string) ShopOperationGenericErrorWithReason {
	return ShopOperationGenericErrorWithReason{mode: mode, reason: reason}
}

func (m ShopOperationGenericErrorWithReason) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationGenericErrorWithReason) String() string {
	return fmt.Sprintf("shop operation generic error with reason mode [%d]", m.mode)
}

func (m ShopOperationGenericErrorWithReason) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(true)
		w.WriteAsciiString(m.reason)
		return w.Bytes()
	}
}

func (m *ShopOperationGenericErrorWithReason) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadBool()
		m.reason = r.ReadAsciiString()
	}
}

// CONFIRM_SHOP_TRANSACTION level-requirement arms. The dispatcher (CShopDlg::
// OnPacket) handles cases 14 (over) and 15 (under) with the SAME wire shape —
// Decode1 mode + Decode4 level. Each mode now gets its OWN discrete struct (no
// shared shape): the structs differ only by the fixed operation key their body
// func resolves. Mode bytes per docs/packets/dispatchers/npc_shop_operation.yaml
// (IDA-verified): OVER_LEVEL_REQUIREMENT=14, UNDER_LEVEL_REQUIREMENT=15
// (version-stable across gms_v83/v84/v87/v95/jms_v185).

// ShopOperationOverLevelRequirement - CONFIRM_SHOP_TRANSACTION OVER_LEVEL_REQUIREMENT arm.
// packet-audit:fname CShopDlg::OnPacket#OverLevelRequirement
type ShopOperationOverLevelRequirement struct {
	mode       byte
	levelLimit uint32
}

func NewShopOperationOverLevelRequirement(mode byte, levelLimit uint32) ShopOperationOverLevelRequirement {
	return ShopOperationOverLevelRequirement{mode: mode, levelLimit: levelLimit}
}

func (m ShopOperationOverLevelRequirement) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationOverLevelRequirement) String() string {
	return fmt.Sprintf("shop operation over level requirement [%d]", m.levelLimit)
}

func (m ShopOperationOverLevelRequirement) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.levelLimit)
		return w.Bytes()
	}
}

func (m *ShopOperationOverLevelRequirement) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.levelLimit = r.ReadUint32()
	}
}

// ShopOperationUnderLevelRequirement - CONFIRM_SHOP_TRANSACTION UNDER_LEVEL_REQUIREMENT arm.
// packet-audit:fname CShopDlg::OnPacket#UnderLevelRequirement
type ShopOperationUnderLevelRequirement struct {
	mode       byte
	levelLimit uint32
}

func NewShopOperationUnderLevelRequirement(mode byte, levelLimit uint32) ShopOperationUnderLevelRequirement {
	return ShopOperationUnderLevelRequirement{mode: mode, levelLimit: levelLimit}
}

func (m ShopOperationUnderLevelRequirement) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationUnderLevelRequirement) String() string {
	return fmt.Sprintf("shop operation under level requirement [%d]", m.levelLimit)
}

func (m ShopOperationUnderLevelRequirement) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.levelLimit)
		return w.Bytes()
	}
}

func (m *ShopOperationUnderLevelRequirement) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.levelLimit = r.ReadUint32()
	}
}

package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
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

// ShopOperationGenericError - mode, hasReason, reason
// packet-audit:fname CShopDlg::OnPacket#GenericError
type ShopOperationGenericError struct {
	mode      byte
	hasReason bool
	reason    string
}

func NewShopOperationGenericError(mode byte) ShopOperationGenericError {
	return ShopOperationGenericError{mode: mode, hasReason: false}
}

func NewShopOperationGenericErrorWithReason(mode byte, reason string) ShopOperationGenericError {
	return ShopOperationGenericError{mode: mode, hasReason: true, reason: reason}
}

func (m ShopOperationGenericError) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationGenericError) String() string {
	return fmt.Sprintf("shop operation generic error mode [%d]", m.mode)
}

func (m ShopOperationGenericError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.hasReason)
		if m.hasReason {
			w.WriteAsciiString(m.reason)
		}
		return w.Bytes()
	}
}

func (m *ShopOperationGenericError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.hasReason = r.ReadBool()
		if m.hasReason {
			m.reason = r.ReadAsciiString()
		}
	}
}

// ShopOperationLevelRequirement - mode, levelLimit
// packet-audit:fname CShopDlg::OnPacket#LevelRequirement
type ShopOperationLevelRequirement struct {
	mode       byte
	levelLimit uint32
}

func NewShopOperationLevelRequirement(mode byte, levelLimit uint32) ShopOperationLevelRequirement {
	return ShopOperationLevelRequirement{mode: mode, levelLimit: levelLimit}
}

func (m ShopOperationLevelRequirement) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationLevelRequirement) String() string {
	return fmt.Sprintf("shop operation level requirement [%d]", m.levelLimit)
}

func (m ShopOperationLevelRequirement) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.levelLimit)
		return w.Bytes()
	}
}

func (m *ShopOperationLevelRequirement) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.levelLimit = r.ReadUint32()
	}
}

package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationWriter = "CashShopOperation"

// OperationError - mode + error code byte
// packet-audit:fname CCashShop::OnCashItemResult#OperationError
type OperationError struct {
	mode      byte
	errorCode byte
}

func NewOperationError(mode byte, errorCode byte) OperationError {
	return OperationError{mode: mode, errorCode: errorCode}
}

func (m OperationError) Mode() byte        { return m.mode }
func (m OperationError) ErrorCode() byte   { return m.errorCode }
func (m OperationError) Operation() string { return CashShopOperationWriter }

func (m OperationError) String() string {
	return fmt.Sprintf("mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m OperationError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *OperationError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// InventoryCapacitySuccess - mode, inventoryType, capacity
// packet-audit:fname CCashShop::OnCashItemResult#InventoryCapacitySuccess
type InventoryCapacitySuccess struct {
	mode          byte
	inventoryType byte
	capacity      uint16
}

func NewInventoryCapacitySuccess(mode byte, inventoryType byte, capacity uint16) InventoryCapacitySuccess {
	return InventoryCapacitySuccess{mode: mode, inventoryType: inventoryType, capacity: capacity}
}

func (m InventoryCapacitySuccess) Mode() byte          { return m.mode }
func (m InventoryCapacitySuccess) InventoryType() byte { return m.inventoryType }
func (m InventoryCapacitySuccess) Capacity() uint16    { return m.capacity }
func (m InventoryCapacitySuccess) Operation() string   { return CashShopOperationWriter }

func (m InventoryCapacitySuccess) String() string {
	return fmt.Sprintf("mode [%d] inventoryType [%d] capacity [%d]", m.mode, m.inventoryType, m.capacity)
}

func (m InventoryCapacitySuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.inventoryType)
		w.WriteShort(m.capacity)
		return w.Bytes()
	}
}

func (m *InventoryCapacitySuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.inventoryType = r.ReadByte()
		m.capacity = r.ReadUint16()
	}
}

// InventoryCapacityFailed - mode, errorCode
// packet-audit:fname CCashShop::OnCashItemResult#InventoryCapacityFailed
type InventoryCapacityFailed struct {
	mode      byte
	errorCode byte
}

func NewInventoryCapacityFailed(mode byte, errorCode byte) InventoryCapacityFailed {
	return InventoryCapacityFailed{mode: mode, errorCode: errorCode}
}

func (m InventoryCapacityFailed) Mode() byte        { return m.mode }
func (m InventoryCapacityFailed) ErrorCode() byte   { return m.errorCode }
func (m InventoryCapacityFailed) Operation() string { return CashShopOperationWriter }

func (m InventoryCapacityFailed) String() string {
	return fmt.Sprintf("mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m InventoryCapacityFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *InventoryCapacityFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// WishListLoad - LOAD_WISHLIST arm: mode + 10 x int32 SNs (DecodeBuffer 40).
// Discrete per-mode struct: fixes the LOAD_WISHLIST operation key; never accepts
// the mode from the caller (the body func resolves it). Wire-identical in shape
// to WishListUpdate but a distinct mode arm (CCashShop::OnCashItemResLoadWishDone).
// packet-audit:fname CCashShop::OnCashItemResult#LOAD_WISHLIST
type WishListLoad struct {
	mode  byte
	items []uint32
}

func NewWishListLoad(mode byte, items []uint32) WishListLoad {
	return WishListLoad{mode: mode, items: items}
}

func (m WishListLoad) Mode() byte        { return m.mode }
func (m WishListLoad) Items() []uint32   { return m.items }
func (m WishListLoad) Operation() string { return CashShopOperationWriter }

func (m WishListLoad) String() string {
	return fmt.Sprintf("mode [%d] items [%v]", m.mode, m.items)
}

func (m WishListLoad) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		for i := 0; i < 10; i++ {
			if i < len(m.items) {
				w.WriteInt(m.items[i])
			} else {
				w.WriteInt(uint32(0))
			}
		}
		return w.Bytes()
	}
}

func (m *WishListLoad) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.items = make([]uint32, 10)
		for i := 0; i < 10; i++ {
			m.items[i] = r.ReadUint32()
		}
	}
}

// WishListUpdate - UPDATE_WISHLIST arm: mode + 10 x int32 SNs (DecodeBuffer 40).
// Discrete per-mode struct: fixes the UPDATE_WISHLIST operation key; never accepts
// the mode from the caller (the body func resolves it). Wire-identical in shape
// to WishListLoad but a distinct mode arm (CCashShop::OnCashItemResSetWishDone).
// packet-audit:fname CCashShop::OnCashItemResult#UPDATE_WISHLIST
type WishListUpdate struct {
	mode  byte
	items []uint32
}

func NewWishListUpdate(mode byte, items []uint32) WishListUpdate {
	return WishListUpdate{mode: mode, items: items}
}

func (m WishListUpdate) Mode() byte        { return m.mode }
func (m WishListUpdate) Items() []uint32   { return m.items }
func (m WishListUpdate) Operation() string { return CashShopOperationWriter }

func (m WishListUpdate) String() string {
	return fmt.Sprintf("mode [%d] items [%v]", m.mode, m.items)
}

func (m WishListUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		for i := 0; i < 10; i++ {
			if i < len(m.items) {
				w.WriteInt(m.items[i])
			} else {
				w.WriteInt(uint32(0))
			}
		}
		return w.Bytes()
	}
}

func (m *WishListUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.items = make([]uint32, 10)
		for i := 0; i < 10; i++ {
			m.items[i] = r.ReadUint32()
		}
	}
}

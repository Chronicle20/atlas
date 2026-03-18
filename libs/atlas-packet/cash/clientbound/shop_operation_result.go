package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CashShopOperationWriter = "CashShopOperation"

// OperationError - mode + error code byte
type OperationError struct {
	mode      byte
	errorCode byte
}

func NewOperationError(mode byte, errorCode byte) OperationError {
	return OperationError{mode: mode, errorCode: errorCode}
}

func (m OperationError) Mode() byte      { return m.mode }
func (m OperationError) ErrorCode() byte  { return m.errorCode }
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
type InventoryCapacitySuccess struct {
	mode          byte
	inventoryType byte
	capacity      uint16
}

func NewInventoryCapacitySuccess(mode byte, inventoryType byte, capacity uint16) InventoryCapacitySuccess {
	return InventoryCapacitySuccess{mode: mode, inventoryType: inventoryType, capacity: capacity}
}

func (m InventoryCapacitySuccess) Mode() byte           { return m.mode }
func (m InventoryCapacitySuccess) InventoryType() byte   { return m.inventoryType }
func (m InventoryCapacitySuccess) Capacity() uint16      { return m.capacity }
func (m InventoryCapacitySuccess) Operation() string     { return CashShopOperationWriter }

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
type InventoryCapacityFailed struct {
	mode      byte
	errorCode byte
}

func NewInventoryCapacityFailed(mode byte, errorCode byte) InventoryCapacityFailed {
	return InventoryCapacityFailed{mode: mode, errorCode: errorCode}
}

func (m InventoryCapacityFailed) Mode() byte      { return m.mode }
func (m InventoryCapacityFailed) ErrorCode() byte  { return m.errorCode }
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

// WishList - mode, items (padded to 10 uint32s)
type WishList struct {
	mode  byte
	items []uint32
}

func NewWishList(mode byte, items []uint32) WishList {
	return WishList{mode: mode, items: items}
}

func (m WishList) Mode() byte       { return m.mode }
func (m WishList) Items() []uint32   { return m.items }
func (m WishList) Operation() string { return CashShopOperationWriter }

func (m WishList) String() string {
	return fmt.Sprintf("mode [%d] items [%v]", m.mode, m.items)
}

func (m WishList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
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

func (m *WishList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.items = make([]uint32, 10)
		for i := 0; i < 10; i++ {
			m.items[i] = r.ReadUint32()
		}
	}
}

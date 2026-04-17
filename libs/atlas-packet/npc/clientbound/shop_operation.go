package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NPCShopOperationWriter = "NPCShopOperation"

// ShopOperationSimple - mode only (OK, OutOfStock, NotEnoughMoney, etc.)
type ShopOperationSimple struct {
	mode byte
}

func NewShopOperationSimple(mode byte) ShopOperationSimple {
	return ShopOperationSimple{mode: mode}
}

func (m ShopOperationSimple) Operation() string { return NPCShopOperationWriter }
func (m ShopOperationSimple) String() string    { return fmt.Sprintf("shop operation mode [%d]", m.mode) }

func (m ShopOperationSimple) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShopOperationSimple) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ShopOperationGenericError - mode, hasReason, reason
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

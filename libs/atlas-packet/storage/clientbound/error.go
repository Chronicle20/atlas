package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const StorageOperationWriter = "StorageOperation"

// ErrorSimple - just mode (covers InventoryFull, NotEnoughMesos, OneOfAKind, etc.)
type ErrorSimple struct {
	mode byte
}

func NewStorageErrorSimple(mode byte) ErrorSimple {
	return ErrorSimple{mode: mode}
}

func (m ErrorSimple) Operation() string { return StorageOperationWriter }
func (m ErrorSimple) String() string    { return fmt.Sprintf("storage error mode [%d]", m.mode) }

func (m ErrorSimple) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ErrorSimple) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// UpdateMeso - mode, slots, currency flag, meso
type UpdateMeso struct {
	mode  byte
	slots byte
	meso  uint32
}

func NewStorageUpdateMeso(mode byte, slots byte, meso uint32) UpdateMeso {
	return UpdateMeso{mode: mode, slots: slots, meso: meso}
}

func (m UpdateMeso) Operation() string { return StorageOperationWriter }
func (m UpdateMeso) String() string {
	return fmt.Sprintf("storage update meso [%d]", m.meso)
}

func (m UpdateMeso) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.slots)
		w.WriteLong(2) // StorageFlagCurrency = 2
		w.WriteInt(m.meso)
		return w.Bytes()
	}
}

func (m *UpdateMeso) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.slots = r.ReadByte()
		_ = r.ReadUint64() // flag
		m.meso = r.ReadUint32()
	}
}

// ErrorMessage - mode, bool(true), message
type ErrorMessage struct {
	mode    byte
	message string
}

func NewStorageErrorMessage(mode byte, message string) ErrorMessage {
	return ErrorMessage{mode: mode, message: message}
}

func (m ErrorMessage) Operation() string { return StorageOperationWriter }
func (m ErrorMessage) String() string    { return "storage error message" }

func (m ErrorMessage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(true)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *ErrorMessage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadBool()
		m.message = r.ReadAsciiString()
	}
}

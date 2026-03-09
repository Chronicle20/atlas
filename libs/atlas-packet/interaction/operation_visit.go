package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationVisit struct {
	serialNumber     uint32
	errorCode        byte
	errorMessage     string
	something        bool
	unk1             int16
	cashSerialNumber uint64
}

func (m OperationVisit) SerialNumber() uint32     { return m.serialNumber }
func (m OperationVisit) ErrorCode() byte           { return m.errorCode }
func (m OperationVisit) ErrorMessage() string      { return m.errorMessage }
func (m OperationVisit) Something() bool           { return m.something }
func (m OperationVisit) Unk1() int16               { return m.unk1 }
func (m OperationVisit) CashSerialNumber() uint64  { return m.cashSerialNumber }

func (m OperationVisit) Operation() string { return "OperationVisit" }

func (m OperationVisit) String() string {
	return fmt.Sprintf("serialNumber [%d] errorCode [%d] errorMessage [%s] something [%t] unk1 [%d] cashSerialNumber [%d]", m.serialNumber, m.errorCode, m.errorMessage, m.something, m.unk1, m.cashSerialNumber)
}

func (m OperationVisit) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.serialNumber)
		w.WriteByte(m.errorCode)
		if m.errorCode != 0 {
			w.WriteAsciiString(m.errorMessage)
		}
		w.WriteBool(m.something)
		if m.something {
			w.WriteInt16(m.unk1)
			w.WriteLong(m.cashSerialNumber)
		}
		return w.Bytes()
	}
}

func (m *OperationVisit) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumber = r.ReadUint32()
		m.errorCode = r.ReadByte()
		if m.errorCode != 0 {
			m.errorMessage = r.ReadAsciiString()
		}
		m.something = r.ReadBool()
		if m.something {
			m.unk1 = r.ReadInt16()
			m.cashSerialNumber = r.ReadUint64()
		}
	}
}

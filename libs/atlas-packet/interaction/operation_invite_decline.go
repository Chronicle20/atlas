package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationInviteDecline struct {
	serialNumber uint32
	errorCode    byte
}

func (m OperationInviteDecline) SerialNumber() uint32 { return m.serialNumber }

func (m OperationInviteDecline) ErrorCode() byte { return m.errorCode }

func (m OperationInviteDecline) Operation() string { return "OperationInviteDecline" }

func (m OperationInviteDecline) String() string {
	return fmt.Sprintf("serialNumber [%d], errorCode [%d]", m.serialNumber, m.errorCode)
}

func (m OperationInviteDecline) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.serialNumber)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *OperationInviteDecline) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.serialNumber = r.ReadUint32()
		m.errorCode = r.ReadByte()
	}
}

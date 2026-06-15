package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AdminLogHandle = "AdminLog"

// AdminLog - CField::SendChatMsgSlash#AdminLog (opcode varies per version).
// Sent by the /-command parser to record an admin-command log line. Body: a
// single string (the log message).
// packet-audit:fname CField::SendChatMsgSlash#AdminLog
type AdminLog struct {
	message string
}

func NewAdminLog(message string) AdminLog {
	return AdminLog{message: message}
}

func (m AdminLog) Message() string { return m.message }

func (m AdminLog) Operation() string {
	return AdminLogHandle
}

func (m AdminLog) String() string {
	return fmt.Sprintf("message [%s]", m.message)
}

func (m AdminLog) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *AdminLog) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
	}
}

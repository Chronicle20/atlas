package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerStatusWriter = "ServerStatus"

type ServerStatus struct {
	status uint16
}

func NewServerStatus(status uint16) ServerStatus {
	return ServerStatus{status: status}
}

func (m ServerStatus) Status() uint16      { return m.status }
func (m ServerStatus) Operation() string   { return ServerStatusWriter }
func (m ServerStatus) String() string      { return fmt.Sprintf("status [%d]", m.status) }

func (m ServerStatus) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.status)
		return w.Bytes()
	}
}

func (m *ServerStatus) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.status = r.ReadUint16()
	}
}

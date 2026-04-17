package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const PingWriter = "Ping"

// Ping - CClientSocket::OnAliveAck (server -> client keepalive)
type Ping struct{}

func (m Ping) Operation() string {
	return PingWriter
}

func (m Ping) String() string {
	return ""
}

func (m Ping) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *Ping) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}

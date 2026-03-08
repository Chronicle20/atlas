package socket

import (
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const PongHandle = "PongHandle"

// Pong - CClientSocket::OnAliveReq
type Pong struct{}

func (m Pong) Operation() string {
	return PongHandle
}

func (m Pong) String() string {
	return ""
}

func (m Pong) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m Pong) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}

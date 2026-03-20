package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerListEndWriter = "ServerListEnd"

type ServerListEnd struct{}

func (m ServerListEnd) Operation() string { return ServerListEndWriter }
func (m ServerListEnd) String() string    { return "" }

func (m ServerListEnd) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0xFF)
		return w.Bytes()
	}
}

func (m *ServerListEnd) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadByte() // 0xFF
	}
}

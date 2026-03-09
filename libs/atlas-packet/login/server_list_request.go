package login

import (
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ServerListRequestHandle = "ServerListRequestHandle"

// ServerListRequest - CLogin::ChangeStepImmediate
type ServerListRequest struct {
}

func (m ServerListRequest) Operation() string {
	return ServerListRequestHandle
}

func (m ServerListRequest) String() string {
	return ""
}

func (m ServerListRequest) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *ServerListRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		return
	}
}

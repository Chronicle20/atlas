package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type InviteRequest struct {
	target string
}

func (m InviteRequest) Target() string { return m.target }

func (m InviteRequest) Operation() string { return "Invite" }

func (m InviteRequest) String() string {
	return fmt.Sprintf("target [%s]", m.target)
}

func (m InviteRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.target)
		return w.Bytes()
	}
}

func (m *InviteRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.target = r.ReadAsciiString()
	}
}

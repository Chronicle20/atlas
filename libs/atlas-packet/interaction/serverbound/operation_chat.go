package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type OperationChat struct {
	updateTime uint32
	message    string
}

func (m OperationChat) UpdateTime() uint32 { return m.updateTime }

func (m OperationChat) Message() string { return m.message }

func (m OperationChat) Operation() string { return "OperationChat" }

func (m OperationChat) String() string {
	return fmt.Sprintf("updateTime [%d], message [%s]", m.updateTime, m.message)
}

func (m OperationChat) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "GMS" && t.MajorVersion() >= 87 {
			w.WriteInt(m.updateTime)
		}
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *OperationChat) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() >= 87 {
			m.updateTime = r.ReadUint32()
		}
		m.message = r.ReadAsciiString()
	}
}

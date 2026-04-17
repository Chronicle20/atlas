package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type SetNotice struct {
	notice string
}

func (m SetNotice) Notice() string { return m.notice }

func (m SetNotice) Operation() string { return "SetNotice" }

func (m SetNotice) String() string {
	return fmt.Sprintf("notice [%s]", m.notice)
}

func (m SetNotice) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.notice)
		return w.Bytes()
	}
}

func (m *SetNotice) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.notice = r.ReadAsciiString()
	}
}

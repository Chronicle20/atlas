package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationDeclineInvite struct {
	fromName   string
	myName     string
	alwaysZero byte
}

func (m OperationDeclineInvite) FromName() string {
	return m.fromName
}

func (m OperationDeclineInvite) MyName() string {
	return m.myName
}

func (m OperationDeclineInvite) AlwaysZero() byte {
	return m.alwaysZero
}

func (m OperationDeclineInvite) Operation() string {
	return "OperationDeclineInvite"
}

func (m OperationDeclineInvite) String() string {
	return fmt.Sprintf("fromName [%s] myName [%s] alwaysZero [%d]", m.fromName, m.myName, m.alwaysZero)
}

func (m OperationDeclineInvite) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.fromName)
		w.WriteAsciiString(m.myName)
		w.WriteByte(m.alwaysZero)
		return w.Bytes()
	}
}

func (m *OperationDeclineInvite) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.fromName = r.ReadAsciiString()
		m.myName = r.ReadAsciiString()
		m.alwaysZero = r.ReadByte()
	}
}

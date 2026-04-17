package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationInvite struct {
	targetCharacter string
}

func (m OperationInvite) TargetCharacter() string {
	return m.targetCharacter
}

func (m OperationInvite) Operation() string {
	return "OperationInvite"
}

func (m OperationInvite) String() string {
	return fmt.Sprintf("targetCharacter [%s]", m.targetCharacter)
}

func (m OperationInvite) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.targetCharacter)
		return w.Bytes()
	}
}

func (m *OperationInvite) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.targetCharacter = r.ReadAsciiString()
	}
}

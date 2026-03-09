package party

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationExpel struct {
	targetCharacterId uint32
}

func (m OperationExpel) TargetCharacterId() uint32 {
	return m.targetCharacterId
}

func (m OperationExpel) Operation() string {
	return "OperationExpel"
}

func (m OperationExpel) String() string {
	return fmt.Sprintf("targetCharacterId [%d]", m.targetCharacterId)
}

func (m OperationExpel) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.targetCharacterId)
		return w.Bytes()
	}
}

func (m *OperationExpel) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.targetCharacterId = r.ReadUint32()
	}
}

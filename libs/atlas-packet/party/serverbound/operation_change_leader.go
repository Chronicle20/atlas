package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationChangeLeader struct {
	targetCharacterId uint32
}

func (m OperationChangeLeader) TargetCharacterId() uint32 {
	return m.targetCharacterId
}

func (m OperationChangeLeader) Operation() string {
	return "OperationChangeLeader"
}

func (m OperationChangeLeader) String() string {
	return fmt.Sprintf("targetCharacterId [%d]", m.targetCharacterId)
}

func (m OperationChangeLeader) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.targetCharacterId)
		return w.Bytes()
	}
}

func (m *OperationChangeLeader) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.targetCharacterId = r.ReadUint32()
	}
}

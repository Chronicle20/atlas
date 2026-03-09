package buddy

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationAccept struct {
	fromCharacterId uint32
}

func (m OperationAccept) FromCharacterId() uint32 {
	return m.fromCharacterId
}

func (m OperationAccept) Operation() string {
	return "OperationAccept"
}

func (m OperationAccept) String() string {
	return fmt.Sprintf("fromCharacterId [%d]", m.fromCharacterId)
}

func (m OperationAccept) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.fromCharacterId)
		return w.Bytes()
	}
}

func (m *OperationAccept) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.fromCharacterId = r.ReadUint32()
	}
}

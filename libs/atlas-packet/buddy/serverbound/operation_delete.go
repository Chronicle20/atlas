package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationDelete struct {
	buddyCharacterId uint32
}

func (m OperationDelete) BuddyCharacterId() uint32 {
	return m.buddyCharacterId
}

func (m OperationDelete) Operation() string {
	return "OperationDelete"
}

func (m OperationDelete) String() string {
	return fmt.Sprintf("buddyCharacterId [%d]", m.buddyCharacterId)
}

func (m OperationDelete) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.buddyCharacterId)
		return w.Bytes()
	}
}

func (m *OperationDelete) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.buddyCharacterId = r.ReadUint32()
	}
}

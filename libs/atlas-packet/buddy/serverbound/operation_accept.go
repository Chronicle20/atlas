package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// packet-audit:fname CField::SendAcceptFriendMsg
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

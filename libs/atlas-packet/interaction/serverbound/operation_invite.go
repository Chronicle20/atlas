package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// packet-audit:fname CField::SendInviteTradingRoomMsg
type OperationInvite struct {
	targetCharacterId uint32
}

func (m OperationInvite) TargetCharacterId() uint32 { return m.targetCharacterId }

func (m OperationInvite) Operation() string { return "OperationInvite" }

func (m OperationInvite) String() string {
	return fmt.Sprintf("targetCharacterId [%d]", m.targetCharacterId)
}

func (m OperationInvite) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.targetCharacterId)
		return w.Bytes()
	}
}

func (m *OperationInvite) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.targetCharacterId = r.ReadUint32()
	}
}

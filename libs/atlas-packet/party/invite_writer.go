package party

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Invite struct {
	mode           byte
	partyId        uint32
	originatorName string
}

func NewInvite(mode byte, partyId uint32, originatorName string) Invite {
	return Invite{mode: mode, partyId: partyId, originatorName: originatorName}
}

func (m Invite) Mode() byte            { return m.mode }
func (m Invite) PartyId() uint32       { return m.partyId }
func (m Invite) OriginatorName() string { return m.originatorName }

func (m Invite) Operation() string {
	return PartyOperationWriter
}

func (m Invite) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d], originatorName [%s]", m.mode, m.partyId, m.originatorName)
}

func (m Invite) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		w.WriteAsciiString(m.originatorName)
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *Invite) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.originatorName = r.ReadAsciiString()
		_ = r.ReadByte() // trailing zero
	}
}

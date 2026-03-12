package party

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Disband struct {
	mode     byte
	partyId  uint32
	targetId uint32
}

func NewDisband(mode byte, partyId uint32, targetId uint32) Disband {
	return Disband{mode: mode, partyId: partyId, targetId: targetId}
}

func (m Disband) Mode() byte      { return m.mode }
func (m Disband) PartyId() uint32 { return m.partyId }
func (m Disband) TargetId() uint32 { return m.targetId }

func (m Disband) Operation() string {
	return PartyOperationWriter
}

func (m Disband) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d], targetId [%d]", m.mode, m.partyId, m.targetId)
}

func (m Disband) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		w.WriteInt(m.targetId)
		w.WriteByte(0)
		w.WriteInt(m.partyId)
		return w.Bytes()
	}
}

func (m *Disband) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.targetId = r.ReadUint32()
		_ = r.ReadByte() // constant 0
		_ = r.ReadUint32() // partyId repeated
	}
}

package party

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type DisbandW struct {
	mode     byte
	partyId  uint32
	targetId uint32
}

func NewDisbandW(mode byte, partyId uint32, targetId uint32) DisbandW {
	return DisbandW{mode: mode, partyId: partyId, targetId: targetId}
}

func (m DisbandW) Mode() byte      { return m.mode }
func (m DisbandW) PartyId() uint32 { return m.partyId }
func (m DisbandW) TargetId() uint32 { return m.targetId }

func (m DisbandW) Operation() string {
	return PartyOperationWriter
}

func (m DisbandW) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d], targetId [%d]", m.mode, m.partyId, m.targetId)
}

func (m DisbandW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
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

func (m *DisbandW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.targetId = r.ReadUint32()
		_ = r.ReadByte() // constant 0
		_ = r.ReadUint32() // partyId repeated
	}
}

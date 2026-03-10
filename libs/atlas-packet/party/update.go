package party

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type UpdateW struct {
	mode     byte
	partyId  uint32
	members  []PartyMember
	leaderId uint32
}

func NewUpdateW(mode byte, partyId uint32, members []PartyMember, leaderId uint32) UpdateW {
	return UpdateW{mode: mode, partyId: partyId, members: members, leaderId: leaderId}
}

func (m UpdateW) Mode() byte             { return m.mode }
func (m UpdateW) PartyId() uint32        { return m.partyId }
func (m UpdateW) Members() []PartyMember { return m.members }
func (m UpdateW) LeaderId() uint32       { return m.leaderId }

func (m UpdateW) Operation() string {
	return PartyOperationWriter
}

func (m UpdateW) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d]", m.mode, m.partyId)
}

func (m UpdateW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		WritePartyData(w, m.members, m.leaderId)
		return w.Bytes()
	}
}

func (m *UpdateW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.members, m.leaderId = ReadPartyData(r)
	}
}

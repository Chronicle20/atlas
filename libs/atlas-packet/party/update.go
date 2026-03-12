package party

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Update struct {
	mode     byte
	partyId  uint32
	members  []PartyMember
	leaderId uint32
}

func NewUpdate(mode byte, partyId uint32, members []PartyMember, leaderId uint32) Update {
	return Update{mode: mode, partyId: partyId, members: members, leaderId: leaderId}
}

func (m Update) Mode() byte             { return m.mode }
func (m Update) PartyId() uint32        { return m.partyId }
func (m Update) Members() []PartyMember { return m.members }
func (m Update) LeaderId() uint32       { return m.leaderId }

func (m Update) Operation() string {
	return PartyOperationWriter
}

func (m Update) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d]", m.mode, m.partyId)
}

func (m Update) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		WritePartyData(w, m.members, m.leaderId)
		return w.Bytes()
	}
}

func (m *Update) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.members, m.leaderId = ReadPartyData(r)
	}
}

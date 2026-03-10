package party

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type JoinW struct {
	mode       byte
	partyId    uint32
	targetName string
	members    []PartyMember
	leaderId   uint32
}

func NewJoinW(mode byte, partyId uint32, targetName string, members []PartyMember, leaderId uint32) JoinW {
	return JoinW{mode: mode, partyId: partyId, targetName: targetName, members: members, leaderId: leaderId}
}

func (m JoinW) Mode() byte             { return m.mode }
func (m JoinW) PartyId() uint32        { return m.partyId }
func (m JoinW) TargetName() string     { return m.targetName }
func (m JoinW) Members() []PartyMember { return m.members }
func (m JoinW) LeaderId() uint32       { return m.leaderId }

func (m JoinW) Operation() string {
	return PartyOperationWriter
}

func (m JoinW) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d], targetName [%s]", m.mode, m.partyId, m.targetName)
}

func (m JoinW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		w.WriteAsciiString(m.targetName)
		WritePartyData(w, m.members, m.leaderId)
		return w.Bytes()
	}
}

func (m *JoinW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.targetName = r.ReadAsciiString()
		m.members, m.leaderId = ReadPartyData(r)
	}
}

package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Join struct {
	mode       byte
	partyId    uint32
	targetName string
	members    []party.PartyMember
	leaderId   uint32
}

func NewJoin(mode byte, partyId uint32, targetName string, members []party.PartyMember, leaderId uint32) Join {
	return Join{mode: mode, partyId: partyId, targetName: targetName, members: members, leaderId: leaderId}
}

func (m Join) Mode() byte             { return m.mode }
func (m Join) PartyId() uint32        { return m.partyId }
func (m Join) TargetName() string     { return m.targetName }
func (m Join) Members() []party.PartyMember { return m.members }
func (m Join) LeaderId() uint32       { return m.leaderId }

func (m Join) Operation() string {
	return PartyOperationWriter
}

func (m Join) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d], targetName [%s]", m.mode, m.partyId, m.targetName)
}

func (m Join) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		w.WriteAsciiString(m.targetName)
		party.WritePartyData(w, m.members, m.leaderId)
		return w.Bytes()
	}
}

func (m *Join) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.targetName = r.ReadAsciiString()
		m.members, m.leaderId = party.ReadPartyData(r)
	}
}

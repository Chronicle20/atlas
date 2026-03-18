package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/party"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Left struct {
	mode       byte
	partyId    uint32
	targetId   uint32
	targetName string
	forced     bool
	members    []party.PartyMember
	leaderId   uint32
}

func NewLeft(mode byte, partyId uint32, targetId uint32, targetName string, forced bool, members []party.PartyMember, leaderId uint32) Left {
	return Left{mode: mode, partyId: partyId, targetId: targetId, targetName: targetName, forced: forced, members: members, leaderId: leaderId}
}

func (m Left) Mode() byte             { return m.mode }
func (m Left) PartyId() uint32        { return m.partyId }
func (m Left) TargetId() uint32       { return m.targetId }
func (m Left) TargetName() string     { return m.targetName }
func (m Left) Forced() bool           { return m.forced }
func (m Left) Members() []party.PartyMember { return m.members }
func (m Left) LeaderId() uint32       { return m.leaderId }

func (m Left) Operation() string {
	return PartyOperationWriter
}

func (m Left) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d], targetId [%d], targetName [%s], forced [%t]", m.mode, m.partyId, m.targetId, m.targetName, m.forced)
}

func (m Left) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		w.WriteInt(m.targetId)
		w.WriteByte(1)
		w.WriteBool(m.forced)
		w.WriteAsciiString(m.targetName)
		party.WritePartyData(w, m.members, m.leaderId)
		return w.Bytes()
	}
}

func (m *Left) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.targetId = r.ReadUint32()
		_ = r.ReadByte() // constant 1
		m.forced = r.ReadBool()
		m.targetName = r.ReadAsciiString()
		m.members, m.leaderId = party.ReadPartyData(r)
	}
}

package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Invite struct {
	mode              byte
	partyId           uint32
	originatorName    string
	originatorJobId   uint32
	originatorLevel   uint32
}

func NewInvite(mode byte, partyId uint32, originatorName string, originatorJobId uint32, originatorLevel uint32) Invite {
	return Invite{mode: mode, partyId: partyId, originatorName: originatorName, originatorJobId: originatorJobId, originatorLevel: originatorLevel}
}

func (m Invite) Mode() byte              { return m.mode }
func (m Invite) PartyId() uint32         { return m.partyId }
func (m Invite) OriginatorName() string  { return m.originatorName }
func (m Invite) OriginatorJobId() uint32 { return m.originatorJobId }
func (m Invite) OriginatorLevel() uint32 { return m.originatorLevel }

func (m Invite) Operation() string {
	return PartyOperationWriter
}

func (m Invite) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d], originatorName [%s], originatorJobId [%d], originatorLevel [%d]", m.mode, m.partyId, m.originatorName, m.originatorJobId, m.originatorLevel)
}

func (m Invite) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	v95plus := (t.Region() == "GMS" && t.MajorVersion() >= 95) || t.Region() == "JMS"
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		w.WriteAsciiString(m.originatorName)
		if v95plus {
			w.WriteInt(m.originatorJobId)
			w.WriteInt(m.originatorLevel)
		}
		w.WriteByte(0) // autoJoinFlag
		return w.Bytes()
	}
}

func (m *Invite) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	v95plus := (t.Region() == "GMS" && t.MajorVersion() >= 95) || t.Region() == "JMS"
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.originatorName = r.ReadAsciiString()
		if v95plus {
			m.originatorJobId = r.ReadUint32()
			m.originatorLevel = r.ReadUint32()
		}
		_ = r.ReadByte() // autoJoinFlag
	}
}

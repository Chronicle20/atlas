package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const BuddyInviteWriter = "BuddyInvite"

type Invite struct {
	mode           byte
	actorId        uint32
	originatorId   uint32
	originatorName string
	jobId          uint32
	level          uint32
}

func NewBuddyInvite(mode byte, actorId uint32, originatorId uint32, originatorName string, jobId uint32, level uint32) Invite {
	return Invite{mode: mode, actorId: actorId, originatorId: originatorId, originatorName: originatorName, jobId: jobId, level: level}
}

func (m Invite) Mode() byte             { return m.mode }
func (m Invite) ActorId() uint32        { return m.actorId }
func (m Invite) OriginatorId() uint32   { return m.originatorId }
func (m Invite) OriginatorName() string { return m.originatorName }
func (m Invite) JobId() uint32          { return m.jobId }
func (m Invite) Level() uint32          { return m.level }
func (m Invite) Operation() string      { return BuddyInviteWriter }

func (m Invite) String() string {
	return fmt.Sprintf("invite from [%s] (%d) job [%d] level [%d] for actor [%d]", m.originatorName, m.originatorId, m.jobId, m.level, m.actorId)
}

func (m Invite) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	// CWvsContext::OnFriendResult case 9 (BuddyInvite) read-order, IDA-verified:
	//   v83 @0xa3f2e8: Decode4 originatorId, DecodeStr originatorName, GW_Friend(39), Decode1 inShop — NO jobId/level.
	//   v87 @0xad7ae5: ... DecodeStr originatorName, Decode4 jobId, Decode4 level, GW_Friend(39), Decode1 inShop.
	//   v95 @0xa12630: same as v87.
	//   JMS185 @0xb2a873: same as v87.
	// jobId/level (inviter's job id + level) go between originatorName and the GW_Friend buffer.
	hasJobLevel := t.Region() != "GMS" || t.MajorVersion() >= 87
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.originatorId)
		w.WriteAsciiString(m.originatorName)
		if hasJobLevel {
			w.WriteInt(m.jobId)
			w.WriteInt(m.level)
		}
		b := model.Buddy{
			FriendId:    m.actorId,
			FriendName:  m.originatorName,
			Flag:        0,
			ChannelId:   channel.Id(0),
			FriendGroup: "Default Group",
		}
		w.WriteByteArray(b.Encode(l, ctx)(options))
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *Invite) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	// hasJobLevel gate: see Encode comment above.
	hasJobLevel := t.Region() != "GMS" || t.MajorVersion() >= 87
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.originatorId = r.ReadUint32()
		m.originatorName = r.ReadAsciiString()
		if hasJobLevel {
			m.jobId = r.ReadUint32()
			m.level = r.ReadUint32()
		}
		m.actorId = r.ReadUint32()
		_ = model.ReadPaddedString(r, 13) // friendName
		_ = r.ReadByte()                  // flag
		_ = r.ReadInt32()                 // channelId
		_ = model.ReadPaddedString(r, 17) // friendGroup
		_ = r.ReadByte()                  // inShop
	}
}

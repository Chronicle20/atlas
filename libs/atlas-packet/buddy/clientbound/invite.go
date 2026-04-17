package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const BuddyInviteWriter = "BuddyInvite"

type Invite struct {
	mode           byte
	actorId        uint32
	originatorId   uint32
	originatorName string
}

func NewBuddyInvite(mode byte, actorId uint32, originatorId uint32, originatorName string) Invite {
	return Invite{mode: mode, actorId: actorId, originatorId: originatorId, originatorName: originatorName}
}

func (m Invite) Mode() byte              { return m.mode }
func (m Invite) ActorId() uint32          { return m.actorId }
func (m Invite) OriginatorId() uint32     { return m.originatorId }
func (m Invite) OriginatorName() string   { return m.originatorName }
func (m Invite) Operation() string        { return BuddyInviteWriter }

func (m Invite) String() string {
	return fmt.Sprintf("invite from [%s] (%d) for actor [%d]", m.originatorName, m.originatorId, m.actorId)
}

func (m Invite) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.originatorId)
		w.WriteAsciiString(m.originatorName)
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

func (m *Invite) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.originatorId = r.ReadUint32()
		m.originatorName = r.ReadAsciiString()
		m.actorId = r.ReadUint32()
		_ = model.ReadPaddedString(r, 13) // friendName
		_ = r.ReadByte()                  // flag
		_ = r.ReadInt32()                 // channelId
		_ = model.ReadPaddedString(r, 17) // friendGroup
		_ = r.ReadByte()                  // inShop
	}
}

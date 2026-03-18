package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterChatGeneralHandle = "CharacterChatGeneralHandle"

type General struct {
	updateTime  uint32
	msg         string
	bOnlyBalloon bool
}

func (m General) UpdateTime() uint32 {
	return m.updateTime
}

func (m General) Msg() string {
	return m.msg
}

func (m General) BOnlyBalloon() bool {
	return m.bOnlyBalloon
}

func (m General) Operation() string {
	return CharacterChatGeneralHandle
}

func (m General) String() string {
	return fmt.Sprintf("msg [%s] updateTime [%d] bOnlyBalloon [%t]", m.msg, m.updateTime, m.bOnlyBalloon)
}

func (m General) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteInt(m.updateTime)
		}
		w.WriteAsciiString(m.msg)
		w.WriteBool(m.bOnlyBalloon)
		return w.Bytes()
	}
}

func (m *General) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			m.updateTime = r.ReadUint32()
		}
		m.msg = r.ReadAsciiString()
		m.bOnlyBalloon = r.ReadBool()
	}
}

package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const PetChatHandle = "PetChatHandle"

type Chat struct {
	petId      uint64
	updateTime uint32
	nType      byte
	nAction    byte
	msg        string
}

func (m Chat) PetId() uint64 {
	return m.petId
}

func (m Chat) UpdateTime() uint32 {
	return m.updateTime
}

func (m Chat) NType() byte {
	return m.nType
}

func (m Chat) NAction() byte {
	return m.nAction
}

func (m Chat) Msg() string {
	return m.msg
}

func (m Chat) Operation() string {
	return PetChatHandle
}

func (m Chat) String() string {
	return fmt.Sprintf("petId [%d] updateTime [%d] nType [%d] nAction [%d] msg [%s]", m.petId, m.updateTime, m.nType, m.nAction, m.msg)
}

func (m Chat) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.petId)
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			w.WriteInt(m.updateTime)
		}
		w.WriteByte(m.nType)
		w.WriteByte(m.nAction)
		w.WriteAsciiString(m.msg)
		return w.Bytes()
	}
}

func (m *Chat) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.petId = r.ReadUint64()
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			m.updateTime = r.ReadUint32()
		}
		m.nType = r.ReadByte()
		m.nAction = r.ReadByte()
		m.msg = r.ReadAsciiString()
	}
}

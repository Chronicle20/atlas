package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PartyInviteRejectHandle = "PartyInviteRejectHandle"

type InviteReject struct {
	unk  byte
	from string
}

func (m InviteReject) Unk() byte {
	return m.unk
}

func (m InviteReject) From() string {
	return m.from
}

func (m InviteReject) Operation() string {
	return PartyInviteRejectHandle
}

func (m InviteReject) String() string {
	return fmt.Sprintf("unk [%d] from [%s]", m.unk, m.from)
}

func (m InviteReject) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.unk)
		w.WriteAsciiString(m.from)
		return w.Bytes()
	}
}

func (m *InviteReject) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.unk = r.ReadByte()
		m.from = r.ReadAsciiString()
	}
}

package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type SetMemberTitle struct {
	targetId uint32
	newTitle byte
}

func (m SetMemberTitle) TargetId() uint32 { return m.targetId }
func (m SetMemberTitle) NewTitle() byte    { return m.newTitle }

func (m SetMemberTitle) Operation() string { return "SetMemberTitle" }

func (m SetMemberTitle) String() string {
	return fmt.Sprintf("targetId [%d] newTitle [%d]", m.targetId, m.newTitle)
}

func (m SetMemberTitle) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.targetId)
		w.WriteByte(m.newTitle)
		return w.Bytes()
	}
}

func (m *SetMemberTitle) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.targetId = r.ReadUint32()
		m.newTitle = r.ReadByte()
	}
}

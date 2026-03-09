package fame

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FameChangeHandle = "FameChangeHandle"

// Change - CUser::SendFameChange
type Change struct {
	targetId uint32
	mode     int8
}

func (m Change) TargetId() uint32 {
	return m.targetId
}

func (m Change) Mode() int8 {
	return m.mode
}

func (m Change) Operation() string {
	return FameChangeHandle
}

func (m Change) String() string {
	return fmt.Sprintf("targetId [%d], mode [%d]", m.targetId, m.mode)
}

func (m Change) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.targetId)
		w.WriteInt8(m.mode)
		return w.Bytes()
	}
}

func (m *Change) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.targetId = r.ReadUint32()
		m.mode = r.ReadInt8()
	}
}

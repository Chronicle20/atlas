package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ContinueConversationSelection struct {
	selection int32
	wide      bool
}

func (m ContinueConversationSelection) Selection() int32 { return m.selection }

func (m ContinueConversationSelection) Operation() string { return "ContinueConversationSelection" }

func (m ContinueConversationSelection) String() string {
	return fmt.Sprintf("selection [%d]", m.selection)
}

func (m ContinueConversationSelection) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if m.wide {
			w.WriteInt32(m.selection)
		} else {
			w.WriteByte(byte(m.selection))
		}
		return w.Bytes()
	}
}

func (m *ContinueConversationSelection) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		if r.Available() >= 4 {
			m.selection = r.ReadInt32()
			m.wide = true
		} else {
			m.selection = int32(r.ReadByte())
			m.wide = false
		}
	}
}

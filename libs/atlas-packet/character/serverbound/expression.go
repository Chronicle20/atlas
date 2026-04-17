package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterExpressionHandle = "CharacterExpressionHandle"

// ExpressionRequest - CUser::SendEmotion
type ExpressionRequest struct {
	emote uint32
}

func (m ExpressionRequest) Emote() uint32 {
	return m.emote
}

func (m ExpressionRequest) Operation() string {
	return CharacterExpressionHandle
}

func (m ExpressionRequest) String() string {
	return fmt.Sprintf("emote [%d]", m.emote)
}

func (m ExpressionRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.emote)
		return w.Bytes()
	}
}

func (m *ExpressionRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.emote = r.ReadUint32()
	}
}

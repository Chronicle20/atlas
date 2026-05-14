package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterExpressionHandle = "CharacterExpressionHandle"

// ExpressionRequest - CWvsContext::SendEmotionChange
// Client sends: Encode4(emotion) + Encode4(duration) + Encode1(bByItemOption)
type ExpressionRequest struct {
	emote         uint32
	duration      int32
	byItemOption  bool
}

func (m ExpressionRequest) Emote() uint32        { return m.emote }
func (m ExpressionRequest) Duration() int32      { return m.duration }
func (m ExpressionRequest) ByItemOption() bool   { return m.byItemOption }

func (m ExpressionRequest) Operation() string {
	return CharacterExpressionHandle
}

func (m ExpressionRequest) String() string {
	return fmt.Sprintf("emote [%d], duration [%d], byItemOption [%v]", m.emote, m.duration, m.byItemOption)
}

func (m ExpressionRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.emote)
		w.WriteInt32(m.duration)
		w.WriteBool(m.byItemOption)
		return w.Bytes()
	}
}

func (m *ExpressionRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.emote = r.ReadUint32()
		m.duration = r.ReadInt32()
		m.byItemOption = r.ReadBool()
	}
}

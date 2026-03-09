package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterExpressionHandle = "CharacterExpressionHandle"

// Expression - CUser::SendEmotion
type Expression struct {
	emote uint32
}

func (m Expression) Emote() uint32 {
	return m.emote
}

func (m Expression) Operation() string {
	return CharacterExpressionHandle
}

func (m Expression) String() string {
	return fmt.Sprintf("emote [%d]", m.emote)
}

func (m Expression) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.emote)
		return w.Bytes()
	}
}

func (m *Expression) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.emote = r.ReadUint32()
	}
}

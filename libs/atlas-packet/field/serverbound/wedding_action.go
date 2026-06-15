package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const WeddingActionHandle = "WeddingAction"

// WeddingAction - CField_Wedding::OnWeddingProgress#Action
// Emitted when the groom/bride confirms a cathedral step. Body: step byte.
type WeddingAction struct {
	step byte
}

func NewWeddingAction(step byte) WeddingAction {
	return WeddingAction{step: step}
}

func (m WeddingAction) Step() byte { return m.step }

func (m WeddingAction) Operation() string {
	return WeddingActionHandle
}

func (m WeddingAction) String() string {
	return fmt.Sprintf("step [%d]", m.step)
}

func (m WeddingAction) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.step)
		return w.Bytes()
	}
}

func (m *WeddingAction) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.step = r.ReadByte()
	}
}

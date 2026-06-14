package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SetQuestClearWriter = "SetQuestClear"

// SetQuestClear is the clientbound CField::OnSetQuestClear packet.
// It carries no payload.
type SetQuestClear struct {
}

func NewSetQuestClear() SetQuestClear {
	return SetQuestClear{}
}

func (m SetQuestClear) Operation() string { return SetQuestClearWriter }
func (m SetQuestClear) String() string {
	return "SetQuestClear"
}

func (m SetQuestClear) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *SetQuestClear) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}

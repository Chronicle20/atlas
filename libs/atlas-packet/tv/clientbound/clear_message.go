package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TvClearMessageWriter = "TvClearMessage"

// TvClearMessage tears down the Maple TV UI. The body is empty — no fields,
// no resolution needed.
type TvClearMessage struct {
}

func NewTvClearMessage() TvClearMessage {
	return TvClearMessage{}
}

func (m TvClearMessage) Operation() string { return TvClearMessageWriter }
func (m TvClearMessage) String() string    { return "clear tv message" }

func (m TvClearMessage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *TvClearMessage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}

package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const StopClockWriter = "StopClock"

// packet-audit:fname CField::OnDestroyClock
type StopClock struct {
}

func NewStopClock() StopClock {
	return StopClock{}
}

func (m StopClock) Operation() string { return StopClockWriter }
func (m StopClock) String() string {
	return "StopClock"
}

func (m StopClock) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *StopClock) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}

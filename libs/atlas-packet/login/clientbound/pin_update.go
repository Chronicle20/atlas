package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PinUpdateWriter = "PinUpdate"

type PinUpdate struct {
	mode byte
}

func NewPinUpdate(mode byte) PinUpdate {
	return PinUpdate{mode: mode}
}

func (m PinUpdate) Mode() byte        { return m.mode }
func (m PinUpdate) Operation() string  { return PinUpdateWriter }
func (m PinUpdate) String() string     { return fmt.Sprintf("mode [%d]", m.mode) }

func (m PinUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *PinUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

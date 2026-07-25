package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const GmEventInstructionsWriter = "GmEventInstructions"

// GmEventInstructions is the clientbound CField::OnDesc packet.
// A single byte index selecting which GM-event instruction text to display.
// packet-audit:fname CField::OnDesc
type GmEventInstructions struct {
	index byte
}

func NewGmEventInstructions(index byte) GmEventInstructions {
	return GmEventInstructions{index: index}
}

func (m GmEventInstructions) Index() byte { return m.index }

func (m GmEventInstructions) Operation() string { return GmEventInstructionsWriter }
func (m GmEventInstructions) String() string {
	return fmt.Sprintf("index [%d]", m.index)
}

func (m GmEventInstructions) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.index)
		return w.Bytes()
	}
}

func (m *GmEventInstructions) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.index = r.ReadByte()
	}
}

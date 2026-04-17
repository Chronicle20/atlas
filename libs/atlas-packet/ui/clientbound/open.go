package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const UiOpenWriter = "UiOpen"

type Open struct {
	windowMode byte
}

func NewUiOpen(windowMode byte) Open {
	return Open{windowMode: windowMode}
}

func (m Open) WindowMode() byte   { return m.windowMode }
func (m Open) Operation() string  { return UiOpenWriter }
func (m Open) String() string     { return fmt.Sprintf("windowMode [%d]", m.windowMode) }

func (m Open) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.windowMode)
		return w.Bytes()
	}
}

func (m *Open) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.windowMode = r.ReadByte()
	}
}

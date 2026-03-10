package ui

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const UiDisableWriter = "UiDisable"

type Disable struct {
	enable bool
}

func NewUiDisable(enable bool) Disable {
	return Disable{enable: enable}
}

func (m Disable) Enable() bool      { return m.enable }
func (m Disable) Operation() string { return UiDisableWriter }
func (m Disable) String() string    { return fmt.Sprintf("enable [%t]", m.enable) }

func (m Disable) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.enable)
		return w.Bytes()
	}
}

func (m *Disable) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.enable = r.ReadBool()
	}
}

package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ScriptProgressWriter = "ScriptProgress"

type ScriptProgress struct {
	message string
}

func NewScriptProgress(message string) ScriptProgress {
	return ScriptProgress{message: message}
}

func (m ScriptProgress) Message() string   { return m.message }
func (m ScriptProgress) Operation() string { return ScriptProgressWriter }
func (m ScriptProgress) String() string    { return fmt.Sprintf("message [%s]", m.message) }

func (m ScriptProgress) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *ScriptProgress) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
	}
}

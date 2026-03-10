package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SetAccountResultWriter = "SetAccountResult"

type SetAccountResult struct {
	gender  byte
	success bool
}

func NewSetAccountResult(gender byte, success bool) SetAccountResult {
	return SetAccountResult{gender: gender, success: success}
}

func (m SetAccountResult) Gender() byte     { return m.gender }
func (m SetAccountResult) Success() bool    { return m.success }
func (m SetAccountResult) Operation() string { return SetAccountResultWriter }
func (m SetAccountResult) String() string {
	return fmt.Sprintf("gender [%d], success [%t]", m.gender, m.success)
}

func (m SetAccountResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.gender)
		w.WriteBool(m.success)
		return w.Bytes()
	}
}

func (m *SetAccountResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.gender = r.ReadByte()
		m.success = r.ReadBool()
	}
}

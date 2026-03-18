package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type AddCharacterError struct {
	code byte
}

func NewAddCharacterError(code byte) AddCharacterError {
	return AddCharacterError{code: code}
}

func (m AddCharacterError) Code() byte        { return m.code }
func (m AddCharacterError) Operation() string  { return AddCharacterEntryWriter }
func (m AddCharacterError) String() string     { return fmt.Sprintf("errorCode [%d]", m.code) }

func (m AddCharacterError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		return w.Bytes()
	}
}

func (m *AddCharacterError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
	}
}

package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const KiteErrorWriter = "SpawnKiteError"

type KiteError struct{}

func NewKiteError() KiteError {
	return KiteError{}
}

func (m KiteError) Operation() string { return KiteErrorWriter }
func (m KiteError) String() string    { return "kite error" }

func (m KiteError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *KiteError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}

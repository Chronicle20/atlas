package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	SpawnKiteError = "SpawnKiteError"
)

func SpawnKiteErrorBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

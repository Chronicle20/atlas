package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const StatChanged = "StatChanged"

func StatChangedBody(l logrus.FieldLogger) func(exclRequestSent bool) BodyProducer {
	// TODO this should transmit stats
	return func(exclRequestSent bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(exclRequestSent)
			w.WriteInt(0)
			return w.Bytes()
		}
	}
}

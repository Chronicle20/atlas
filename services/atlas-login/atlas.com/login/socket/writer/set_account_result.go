package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SetAccountResult = "SetAccountResult"

func SetAccountResultBody(l logrus.FieldLogger) func(gender byte) BodyProducer {
	return func(gender byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(gender)
			w.WriteByte(1) // 0 gives you a try again
			return w.Bytes()
		}
	}
}

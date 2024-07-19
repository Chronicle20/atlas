package writer

import (
	"atlas-login/world"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerStatus = "ServerStatus"

func ServerStatusBody(l logrus.FieldLogger) func(status world.Status) BodyProducer {
	return func(status world.Status) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteShort(uint16(status))
			return w.Bytes()
		}
	}
}

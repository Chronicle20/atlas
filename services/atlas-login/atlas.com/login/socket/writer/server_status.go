package writer

import (
	"atlas-login/world"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerStatus = "ServerStatus"

func ServerStatusBody(l logrus.FieldLogger) func(status world.Status) BodyProducer {
	return func(status world.Status) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteShort(uint16(status))
			return w.Bytes()
		}
	}
}

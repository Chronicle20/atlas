package writer

import (
	"atlas-login/world"

	"github.com/Chronicle20/atlas-socket/response"
)

const ServerStatus = "ServerStatus"

func ServerStatusBody(status world.Status) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteShort(uint16(status))
		return w.Bytes()
	}
}

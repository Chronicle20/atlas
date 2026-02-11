package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const Ping = "Ping"

func PingBody() BodyProducer {
	return func(w *response.Writer, _ map[string]interface{}) []byte {
		return w.Bytes()
	}
}

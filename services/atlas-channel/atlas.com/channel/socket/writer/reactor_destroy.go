package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const (
	ReactorDestroy = "ReactorDestroy"
)

func ReactorDestroyBody() func(id uint32, state int8, x int16, y int16) BodyProducer {
	return func(id uint32, state int8, x int16, y int16) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(id)
			w.WriteInt8(state)
			w.WriteInt16(x)
			w.WriteInt16(y)
			return w.Bytes()
		}
	}
}

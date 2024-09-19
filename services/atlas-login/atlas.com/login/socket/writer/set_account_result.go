package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const SetAccountResult = "SetAccountResult"

func SetAccountResultBody(gender byte, success bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(gender)
		w.WriteBool(success)
		return w.Bytes()
	}
}

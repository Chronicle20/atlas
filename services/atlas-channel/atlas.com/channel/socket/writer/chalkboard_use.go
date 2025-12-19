package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const ChalkboardUse = "ChalkboardUse"

func ChalkboardUseBody(characterId uint32, message string) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(characterId)
		w.WriteBool(true)
		w.WriteAsciiString(message)
		return w.Bytes()
	}
}

func ChalkboardClearBody(characterId uint32) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(characterId)
		w.WriteBool(false)
		return w.Bytes()
	}
}

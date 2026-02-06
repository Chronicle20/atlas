package writer

import "github.com/Chronicle20/atlas-socket/response"

const (
	ScriptProgress = "ScriptProgress"
)

func ScriptProgressBody(message string) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteAsciiString(message)
		return w.Bytes()
	}
}

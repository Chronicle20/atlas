package writer

import "github.com/Chronicle20/atlas-socket/response"

const CharacterSitResult = "CharacterSitResult"

func CharacterSitBody(chairId uint16) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(1)
		w.WriteShort(chairId)
		return w.Bytes()
	}
}

func CharacterCancelSitBody() BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(0)
		return w.Bytes()
	}
}

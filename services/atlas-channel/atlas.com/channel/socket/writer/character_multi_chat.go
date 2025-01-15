package writer

import "github.com/Chronicle20/atlas-socket/response"

const CharacterMultiChat = "CharacterMultiChat"

func CharacterMultiChatBody(from string, message string, mode byte) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(mode)
		w.WriteAsciiString(from)
		w.WriteAsciiString(message)
		return w.Bytes()
	}
}

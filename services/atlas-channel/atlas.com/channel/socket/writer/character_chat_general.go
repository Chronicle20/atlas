package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const CharacterChatGeneral = "CharacterChatGeneral"

func CharacterChatGeneralBody(fromCharacterId uint32, gm bool, message string, show bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(fromCharacterId)
		w.WriteBool(gm)
		w.WriteAsciiString(message)
		w.WriteBool(show)
		return w.Bytes()
	}
}

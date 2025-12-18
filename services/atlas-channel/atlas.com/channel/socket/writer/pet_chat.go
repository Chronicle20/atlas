package writer

import (
	"atlas-channel/pet"
	"github.com/Chronicle20/atlas-socket/response"
)

const PetChat = "PetChat"

func PetChatBody(p pet.Model, nType byte, nAction byte, message string, balloon bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(p.OwnerId())
		w.WriteInt8(p.Slot())
		w.WriteByte(nType)
		w.WriteByte(nAction)
		w.WriteAsciiString(message)
		w.WriteBool(balloon)
		return w.Bytes()
	}
}

package writer

import (
	"atlas-channel/pet"
	"github.com/Chronicle20/atlas-socket/response"
)

const PetCommandResponse = "PetCommandResponse"

func PetCommandResponseBody(p pet.Model, animation byte, success bool, balloon bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(p.OwnerId())
		w.WriteInt8(p.Slot())
		w.WriteByte(0) // TODO 1 if this is a closeness increasing item??
		w.WriteByte(animation)
		w.WriteBool(success)
		w.WriteBool(balloon)
		return w.Bytes()
	}
}

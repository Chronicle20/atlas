package writer

import (
	"atlas-channel/pet"
	"github.com/Chronicle20/atlas-socket/response"
)

const PetExcludeResponse = "PetExcludeResponse"

func PetExcludeResponseBody(p pet.Model) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(p.OwnerId())
		w.WriteInt8(p.Slot())
		w.WriteLong(uint64(p.Id()))
		w.WriteByte(byte(len(p.Excludes())))
		for _, e := range p.Excludes() {
			w.WriteInt(e.ItemId())
		}
		return w.Bytes()
	}
}

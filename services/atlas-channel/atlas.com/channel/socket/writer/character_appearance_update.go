package writer

import (
	"atlas-channel/character"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
)

const CharacterAppearanceUpdate = "CharacterAppearanceUpdate"

func CharacterAppearanceUpdateBody(tenant tenant.Model) func(c character.Model) BodyProducer {
	return func(c character.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(c.Id())
			w.WriteByte(1) // mode, 1, 2, 4
			WriteCharacterLook(tenant)(w, c, false)
			w.WriteByte(0) // crush ring
			w.WriteByte(0) // friendship ring
			w.WriteByte(0) // marriage ring
			w.WriteInt(0)  // completed set item id
			return w.Bytes()
		}
	}
}

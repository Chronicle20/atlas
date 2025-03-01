package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const PartyMemberHP = "PartyMemberHP"

func PartyMemberHPBody(characterId uint32, hp uint16, maxHp uint16) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(characterId)
		w.WriteInt(uint32(hp))
		w.WriteInt(uint32(maxHp))
		return w.Bytes()
	}
}

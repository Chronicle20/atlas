package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const CharacterDespawn = "CharacterDespawn"

func CharacterDespawnBody(characterId uint32) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(characterId)
		return w.Bytes()
	}
}

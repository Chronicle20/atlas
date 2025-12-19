package writer

import "github.com/Chronicle20/atlas-socket/response"

const CharacterShowChair = "CharacterShowChair"

func CharacterShowChairBody(characterId uint32, chairId uint32) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(characterId)
		w.WriteInt(chairId)
		return w.Bytes()
	}
}

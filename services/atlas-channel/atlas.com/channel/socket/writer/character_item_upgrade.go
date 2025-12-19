package writer

import "github.com/Chronicle20/atlas-socket/response"

const CharacterItemUpgrade = "CharacterItemUpgrade"

func CharacterItemUpgradeBody(characterId uint32, success bool, cursed bool, legendarySpirit bool, whiteScroll bool) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt(characterId)
		w.WriteBool(success)
		w.WriteBool(cursed)
		w.WriteBool(legendarySpirit)
		w.WriteBool(whiteScroll)
		return w.Bytes()
	}
}

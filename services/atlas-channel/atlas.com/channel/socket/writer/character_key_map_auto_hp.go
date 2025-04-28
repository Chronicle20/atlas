package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const CharacterKeyMapAutoHp = "CharacterKeyMapAutoHp"

func CharacterKeyMapAutoHpBody(action int32) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteInt32(action)
		return w.Bytes()
	}
}

package writer

import (
	"atlas-channel/character/key"
	"github.com/Chronicle20/atlas-socket/response"
)

const CharacterKeyMap = "CharacterKeyMap"

func CharacterKeyMapBody(keys map[int32]key.Model) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(0)
		for i := range 90 {
			if val, ok := keys[int32(i)]; ok {
				w.WriteInt8(val.Type())
				w.WriteInt32(val.Action())
			} else {
				w.WriteInt8(0)
				w.WriteInt32(0)
			}
		}
		return w.Bytes()
	}
}

func CharacterKeyMapResetToDefaultBody() BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(1)
		return w.Bytes()
	}
}

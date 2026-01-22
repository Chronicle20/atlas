package writer

import "github.com/Chronicle20/atlas-socket/response"

const CharacterHint = "CharacterHint"

func CharacterHintBody(hint string, width uint16, height uint16, atPoint bool, x int32, y int32) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		if width < 1 {
			width = uint16(len(hint)) * 10
			if width < 40 {
				width = 40
			}
		}
		if height < 5 {
			height = 5
		}
		w.WriteAsciiString(hint)
		w.WriteShort(width)
		w.WriteShort(height)
		w.WriteBool(!atPoint)
		if atPoint {
			w.WriteInt32(x)
			w.WriteInt32(y)
		}
		return w.Bytes()
	}
}

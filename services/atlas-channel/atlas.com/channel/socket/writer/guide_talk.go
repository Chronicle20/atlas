package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	GuideTalk = "GuideTalk"
)

func GuideTalkBody(_ logrus.FieldLogger) func(message string, hintId uint32, duration uint32) BodyProducer {
	return func(message string, hintId uint32, duration uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// default duration is 7000 (ms?)

			if len(message) == 0 {
				w.WriteBool(false)
				w.WriteInt(hintId)
				w.WriteInt(duration)
			} else {
				w.WriteBool(true)
				w.WriteAsciiString(message)
				w.WriteInt(hintId)
				w.WriteInt(duration)
			}
			return w.Bytes()
		}
	}
}

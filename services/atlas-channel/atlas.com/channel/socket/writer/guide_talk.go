package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	GuideTalk = "GuideTalk"
)

func GuideTalkMessageBody(_ logrus.FieldLogger) func(message string, width uint32, duration uint32) BodyProducer {
	return func(message string, width uint32, duration uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// default width is 200
			// default duration is 4000 (ms?)
			w.WriteBool(true)
			w.WriteAsciiString(message)
			w.WriteInt(width)
			w.WriteInt(duration)
			return w.Bytes()
		}
	}
}

func GuideTalkIdxBody(_ logrus.FieldLogger) func(hintId uint32, duration uint32) BodyProducer {
	return func(hintId uint32, duration uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// default duration is 7000 (ms?)
			w.WriteBool(false)
			w.WriteInt(hintId)
			w.WriteInt(duration)
			return w.Bytes()
		}
	}
}

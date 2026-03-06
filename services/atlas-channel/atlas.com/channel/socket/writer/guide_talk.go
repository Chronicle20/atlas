package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	GuideTalk = "GuideTalk"
)

func GuideTalkMessageBody(message string, width uint32, duration uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
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

func GuideTalkIdxBody(hintId uint32, duration uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			// default duration is 7000 (ms?)
			w.WriteBool(false)
			w.WriteInt(hintId)
			w.WriteInt(duration)
			return w.Bytes()
		}
	}
}

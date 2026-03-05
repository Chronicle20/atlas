package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	MiniRoom = "MiniRoom"
)

func MiniRoomBody(l logrus.FieldLogger) func() BodyProducer {
	return func() BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			return w.Bytes()
		}
	}
}

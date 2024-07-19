package writer

import (
	"atlas-login/world"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerListRecommendations = "ServerListRecommendations"

func ServerListRecommendationsBody(l logrus.FieldLogger) func(wrs []world.Recommendation) BodyProducer {
	return func(wrs []world.Recommendation) BodyProducer {
		return func(w *response.Writer, _ map[string]interface{}) []byte {
			w.WriteByte(byte(len(wrs)))
			for _, x := range wrs {
				w.WriteInt(uint32(x.WorldId()))
				w.WriteAsciiString(x.Reason())
			}
			rtn := w.Bytes()
			return rtn
		}
	}
}

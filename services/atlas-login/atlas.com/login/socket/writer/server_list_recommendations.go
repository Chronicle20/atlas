package writer

import (
	"atlas-login/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerListRecommendations = "ServerListRecommendations"

func ServerListRecommendationsBody(wrs []model.Recommendation) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(len(wrs)))
			for _, x := range wrs {
				w.WriteByteArray(x.Encode(l, ctx)(options))
				w.WriteInt(uint32(x.WorldId()))
				w.WriteAsciiString(x.Reason())
			}
			rtn := w.Bytes()
			return rtn
		}
	}
}

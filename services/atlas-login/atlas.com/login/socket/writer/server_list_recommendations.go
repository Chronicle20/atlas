package writer

import (
	"atlas-login/socket/model"
	"context"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const ServerListRecommendations = "ServerListRecommendations"

func ServerListRecommendationsBody(l logrus.FieldLogger, ctx context.Context) func(wrs []model.Recommendation) BodyProducer {
	return func(wrs []model.Recommendation) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(len(wrs)))
			for _, x := range wrs {
				_ = x.Encode(l, tenant.MustFromContext(ctx), options)
				w.WriteInt(uint32(x.WorldId()))
				w.WriteAsciiString(x.Reason())
			}
			rtn := w.Bytes()
			return rtn
		}
	}
}

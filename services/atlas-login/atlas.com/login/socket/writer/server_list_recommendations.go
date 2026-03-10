package writer

import (
	"atlas-login/socket/model"
	"context"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const ServerListRecommendations = "ServerListRecommendations"

func ServerListRecommendationsBody(wrs []model.Recommendation) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			prs := make([]packetmodel.WorldRecommendation, len(wrs))
			for i, x := range wrs {
				prs[i] = packetmodel.NewWorldRecommendation(x.WorldId(), x.Reason())
			}
			return loginpkt.NewServerListRecommendations(prs).Encode(l, ctx)(options)
		}
	}
}

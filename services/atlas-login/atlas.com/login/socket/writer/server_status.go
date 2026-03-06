package writer

import (
	"atlas-login/world"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerStatus = "ServerStatus"

func ServerStatusBody(status world.Status) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteShort(uint16(status))
			return w.Bytes()
		}
	}
}

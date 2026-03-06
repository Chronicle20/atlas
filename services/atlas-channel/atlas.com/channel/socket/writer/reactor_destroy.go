package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	ReactorDestroy = "ReactorDestroy"
)

func ReactorDestroyBody(id uint32, state int8, x int16, y int16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(id)
			w.WriteInt8(state)
			w.WriteInt16(x)
			w.WriteInt16(y)
			return w.Bytes()
		}
	}
}

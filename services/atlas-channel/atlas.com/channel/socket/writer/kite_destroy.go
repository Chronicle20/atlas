package writer

import (
	"atlas-channel/kite"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type KiteDestroyAnimationType byte

const (
	DestroyKite               = "DestroyKite"
	KiteDestroyAnimationType1 = KiteDestroyAnimationType(0)
	KiteDestroyAnimationType2 = KiteDestroyAnimationType(1)
)

func DestroyKiteBody(m kite.Model, animationType KiteDestroyAnimationType) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(animationType))
			w.WriteInt(m.Id())
			return w.Bytes()
		}
	}
}

package writer

import (
	"atlas-channel/kite"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type KiteDestroyAnimationType byte

const (
	DestroyKite               = "DestroyKite"
	KiteDestroyAnimationType1 = KiteDestroyAnimationType(0)
	KiteDestroyAnimationType2 = KiteDestroyAnimationType(1)
)

func DestroyKiteBody(l logrus.FieldLogger) func(m kite.Model, animationType KiteDestroyAnimationType) BodyProducer {
	return func(m kite.Model, animationType KiteDestroyAnimationType) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(byte(animationType))
			w.WriteInt(m.Id())
			return w.Bytes()
		}
	}
}

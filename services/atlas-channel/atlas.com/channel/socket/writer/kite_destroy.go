package writer

import (
	"atlas-channel/kite"

	fieldpkt "github.com/Chronicle20/atlas-packet/field"
	"github.com/Chronicle20/atlas-socket/packet"
)

type KiteDestroyAnimationType byte

const (
	DestroyKite               = "DestroyKite"
	KiteDestroyAnimationType1 = KiteDestroyAnimationType(0)
	KiteDestroyAnimationType2 = KiteDestroyAnimationType(1)
)

func DestroyKiteBody(m kite.Model, animationType KiteDestroyAnimationType) packet.Encode {
	return fieldpkt.NewKiteDestroy(m.Id(), fieldpkt.KiteDestroyAnimationType(animationType)).Encode
}

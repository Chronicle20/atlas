package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// SnowballTouchBody builds the LEFT_KNOCK_BACK (CField_SnowBall::OnSnowBallTouch)
// clientbound packet, which carries an empty body. The codec + route exist so the
// knockback feature can be switched on later without another packet-plumbing pass;
// the uncalled writer is a documented seam (IMPLEMENTING_A_PACKET D2), not dead code.
func SnowballTouchBody() packet.Encode {
	return fieldcb.NewSnowballTouch().Encode
}

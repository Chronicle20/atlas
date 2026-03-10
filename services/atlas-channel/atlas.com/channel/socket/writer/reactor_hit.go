package writer

import (
	"atlas-channel/reactor"

	reactorpkt "github.com/Chronicle20/atlas-packet/reactor"
	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	ReactorHit = "ReactorHit"
)

func ReactorHitBody(m reactor.Model) packet.Encode {
	return reactorpkt.NewReactorHitW(m.Id(), m.State(), m.X(), m.Y(), uint16(m.Direction())).Encode
}

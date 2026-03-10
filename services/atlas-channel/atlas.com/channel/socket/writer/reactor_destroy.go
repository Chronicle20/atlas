package writer

import (
	reactorpkt "github.com/Chronicle20/atlas-packet/reactor"
	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	ReactorDestroy = "ReactorDestroy"
)

func ReactorDestroyBody(id uint32, state int8, x int16, y int16) packet.Encode {
	return reactorpkt.NewReactorDestroy(id, state, x, y).Encode
}

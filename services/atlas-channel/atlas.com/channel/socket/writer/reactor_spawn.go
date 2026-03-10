package writer

import (
	"atlas-channel/reactor"

	reactorpkt "github.com/Chronicle20/atlas-packet/reactor"
	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	ReactorSpawn = "ReactorSpawn"
)

func ReactorSpawnBody(m reactor.Model) packet.Encode {
	return reactorpkt.NewReactorSpawn(m.Id(), m.Classification(), m.State(), m.X(), m.Y(), m.Direction(), m.Name()).Encode
}

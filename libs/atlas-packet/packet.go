package atlas_packet

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

type Packet interface {
	Operation() string
	fmt.Stringer
	packet.Encoder
	packet.Decoder
}

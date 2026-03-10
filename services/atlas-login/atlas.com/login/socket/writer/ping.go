package writer

import (
	"github.com/Chronicle20/atlas-socket/packet"

	socketpkt "github.com/Chronicle20/atlas-packet/socket"
)

const Ping = "Ping"

func PingBody() packet.Encode {
	return socketpkt.Ping{}.Encode
}

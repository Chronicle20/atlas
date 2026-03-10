package writer

import (
	"github.com/Chronicle20/atlas-packet/socket"
	"github.com/Chronicle20/atlas-socket/packet"
)

const Ping = "Ping"

func PingBody() packet.Encode {
	return socket.Ping{}.Encode
}

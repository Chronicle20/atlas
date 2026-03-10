package writer

import (
	"atlas-login/world"

	"github.com/Chronicle20/atlas-socket/packet"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
)

const ServerStatus = "ServerStatus"

func ServerStatusBody(status world.Status) packet.Encode {
	return loginpkt.NewServerStatus(uint16(status)).Encode
}

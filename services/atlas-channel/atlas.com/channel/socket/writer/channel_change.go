package writer

import (
	channelpkt "github.com/Chronicle20/atlas-packet/channel"
	"github.com/Chronicle20/atlas-socket/packet"
)

const ChannelChange = "ChannelChange"

func ChannelChangeBody(ipAddr string, port uint16) packet.Encode {
	return channelpkt.NewChannelChangeW(ipAddr, port).Encode
}

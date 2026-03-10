package writer

import (
	"github.com/Chronicle20/atlas-socket/packet"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
)

const SelectWorld = "SelectWorld"

func SelectWorldBody(worldId int) packet.Encode {
	return loginpkt.NewSelectWorld(uint32(worldId)).Encode
}

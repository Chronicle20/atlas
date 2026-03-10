package writer

import (
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const ChalkboardUse = "ChalkboardUse"

func ChalkboardUseBody(characterId uint32, message string) packet.Encode {
	return charpkt.NewChalkboardUse(characterId, message).Encode
}

func ChalkboardClearBody(characterId uint32) packet.Encode {
	return charpkt.NewChalkboardClear(characterId).Encode
}

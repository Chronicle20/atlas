package writer

import (
	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterDespawn = "CharacterDespawn"

func CharacterDespawnBody(characterId uint32) packet.Encode {
	return charpkt.NewCharacterDespawn(characterId).Encode
}

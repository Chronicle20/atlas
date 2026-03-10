package writer

import (
	"github.com/Chronicle20/atlas-packet/model"

	charpkt "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/packet"
)

const CharacterMovement = "CharacterMovement"

func CharacterMovementBody(characterId uint32, movement model.Movement) packet.Encode {
	return charpkt.NewCharacterMovementW(characterId, movement).Encode
}

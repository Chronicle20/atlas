package writer

import (
	"atlas-channel/pet"

	"github.com/Chronicle20/atlas-packet/model"
	petpkt "github.com/Chronicle20/atlas-packet/pet"
	"github.com/Chronicle20/atlas-socket/packet"
)

const PetMovement = "PetMovement"

func PetMovementBody(p pet.Model, movement model.Movement) packet.Encode {
	return petpkt.NewPetMovementW(p.OwnerId(), p.Slot(), movement).Encode
}

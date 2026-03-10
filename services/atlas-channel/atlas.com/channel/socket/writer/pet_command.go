package writer

import (
	"atlas-channel/pet"

	petpkt "github.com/Chronicle20/atlas-packet/pet"
	"github.com/Chronicle20/atlas-socket/packet"
)

const PetCommandResponse = "PetCommandResponse"

func PetCommandResponseBody(p pet.Model, animation byte, success bool, balloon bool) packet.Encode {
	return petpkt.NewPetCommandResponse(p.OwnerId(), p.Slot(), animation, success, balloon).Encode
}

func PetFoodResponseBody(p pet.Model, animation byte, success bool, balloon bool) packet.Encode {
	return petpkt.NewPetFoodResponse(p.OwnerId(), p.Slot(), animation, success, balloon).Encode
}

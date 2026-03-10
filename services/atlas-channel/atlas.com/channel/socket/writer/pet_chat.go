package writer

import (
	"atlas-channel/pet"

	petpkt "github.com/Chronicle20/atlas-packet/pet"
	"github.com/Chronicle20/atlas-socket/packet"
)

const PetChat = "PetChat"

func PetChatBody(p pet.Model, nType byte, nAction byte, message string, balloon bool) packet.Encode {
	return petpkt.NewPetChatW(p.OwnerId(), p.Slot(), nType, nAction, message, balloon).Encode
}

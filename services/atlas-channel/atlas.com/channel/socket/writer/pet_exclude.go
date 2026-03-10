package writer

import (
	"atlas-channel/pet"

	petpkt "github.com/Chronicle20/atlas-packet/pet"
	"github.com/Chronicle20/atlas-socket/packet"
)

const PetExcludeResponse = "PetExcludeResponse"

func PetExcludeResponseBody(p pet.Model) packet.Encode {
	excludeIds := make([]uint32, len(p.Excludes()))
	for i, e := range p.Excludes() {
		excludeIds[i] = e.ItemId()
	}
	return petpkt.NewPetExcludeResponse(p.OwnerId(), p.Slot(), uint64(p.Id()), excludeIds).Encode
}

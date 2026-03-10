package writer

import (
	petpkt "github.com/Chronicle20/atlas-packet/pet"
	"github.com/Chronicle20/atlas-socket/packet"
)

const PetCashFoodResult = "PetCashFoodResult"

func PetCashFoodErrorResultBody() packet.Encode {
	return petpkt.NewPetCashFoodResultError().Encode
}

func PetCashFoodResultBody(failure bool, index byte) packet.Encode {
	if failure {
		return petpkt.NewPetCashFoodResultError().Encode
	}
	return petpkt.NewPetCashFoodResult(index).Encode
}

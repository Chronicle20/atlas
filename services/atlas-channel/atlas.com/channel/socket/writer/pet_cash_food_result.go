package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const PetCashFoodResult = "PetCashFoodResult"

func PetCashFoodErrorResultBody() BodyProducer {
	return PetCashFoodResultBody(true, 0)
}

func PetCashFoodResultBody(failure bool, index byte) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteBool(failure)
		if !failure {
			w.WriteByte(index)
		}
		return w.Bytes()
	}
}

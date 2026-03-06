package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetCashFoodResult = "PetCashFoodResult"

func PetCashFoodErrorResultBody() packet.Encode {
	return PetCashFoodResultBody(true, 0)
}

func PetCashFoodResultBody(failure bool, index byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteBool(failure)
			if !failure {
				w.WriteByte(index)
			}
			return w.Bytes()
		}
	}
}

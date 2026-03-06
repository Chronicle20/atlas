package writer

import (
	"atlas-channel/pet"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetCommandResponse = "PetCommandResponse"

func PetCommandResponseBody(p pet.Model, animation byte, success bool, balloon bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(p.OwnerId())
			w.WriteInt8(p.Slot())
			w.WriteByte(0)
			w.WriteByte(animation)
			w.WriteBool(success)
			w.WriteBool(balloon)
			return w.Bytes()
		}
	}
}

func PetFoodResponseBody(p pet.Model, animation byte, success bool, balloon bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(p.OwnerId())
			w.WriteInt8(p.Slot())
			w.WriteByte(1)
			w.WriteByte(animation)
			w.WriteBool(success)
			w.WriteBool(balloon)
			return w.Bytes()
		}
	}
}

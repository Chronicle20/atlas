package writer

import (
	"atlas-channel/pet"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetChat = "PetChat"

func PetChatBody(p pet.Model, nType byte, nAction byte, message string, balloon bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(p.OwnerId())
			w.WriteInt8(p.Slot())
			w.WriteByte(nType)
			w.WriteByte(nAction)
			w.WriteAsciiString(message)
			w.WriteBool(balloon)
			return w.Bytes()
		}
	}
}

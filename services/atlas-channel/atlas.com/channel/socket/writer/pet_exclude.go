package writer

import (
	"atlas-channel/pet"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetExcludeResponse = "PetExcludeResponse"

func PetExcludeResponseBody(p pet.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(p.OwnerId())
			w.WriteInt8(p.Slot())
			w.WriteLong(uint64(p.Id()))
			w.WriteByte(byte(len(p.Excludes())))
			for _, e := range p.Excludes() {
				w.WriteInt(e.ItemId())
			}
			return w.Bytes()
		}
	}
}

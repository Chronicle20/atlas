package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PartyMemberHP = "PartyMemberHP"

func PartyMemberHPBody(characterId uint32, hp uint16, maxHp uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteInt(uint32(hp))
			w.WriteInt(uint32(maxHp))
			return w.Bytes()
		}
	}
}

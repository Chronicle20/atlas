package writer

import (
	"context"

	partypkt "github.com/Chronicle20/atlas-packet/party"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const PartyMemberHP = "PartyMemberHP"

func PartyMemberHPBody(characterId uint32, hp uint16, maxHp uint16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return partypkt.NewPartyMemberHP(characterId, hp, maxHp).Encode(l, ctx)
	}
}

package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterItemUpgrade = "CharacterItemUpgrade"

func CharacterItemUpgradeBody(characterId uint32, success bool, cursed bool, legendarySpirit bool, whiteScroll bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteBool(success)
			w.WriteBool(cursed)
			w.WriteBool(legendarySpirit)
			w.WriteBool(whiteScroll)
			return w.Bytes()
		}
	}
}

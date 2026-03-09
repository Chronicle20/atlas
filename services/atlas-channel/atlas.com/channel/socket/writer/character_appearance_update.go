package writer

import (
	"atlas-channel/character"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterAppearanceUpdate = "CharacterAppearanceUpdate"

func CharacterAppearanceUpdateBody(c character.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(c.Id())
			w.WriteByte(1) // mode, 1, 2, 4
			ava := model.NewFromCharacter(c, false)
			w.WriteByteArray(ava.Encode(l, ctx)(options))
			w.WriteByte(0) // crush ring
			w.WriteByte(0) // friendship ring
			w.WriteByte(0) // marriage ring
			w.WriteInt(0)  // completed set item id
			return w.Bytes()
		}
	}
}

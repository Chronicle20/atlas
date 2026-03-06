package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	GuildEmblemChanged = "GuildEmblemChanged"
)

func ForeignGuildEmblemChangedBody(characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteShort(logoBackground)
			w.WriteByte(logoBackgroundColor)
			w.WriteShort(logo)
			w.WriteByte(logoColor)
			return w.Bytes()
		}
	}
}

package writer

import (
	"context"

	guildpkt "github.com/Chronicle20/atlas-packet/guild"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	GuildEmblemChanged = "GuildEmblemChanged"
)

func ForeignGuildEmblemChangedBody(characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return guildpkt.NewForeignEmblemChanged(characterId, logo, logoColor, logoBackground, logoBackgroundColor).Encode(l, ctx)
	}
}

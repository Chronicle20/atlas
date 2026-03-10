package writer

import (
	"context"

	guildpkt "github.com/Chronicle20/atlas-packet/guild"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	GuildNameChanged = "GuildNameChanged"
)

func ForeignGuildNameChangedBody(characterId uint32, name string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return guildpkt.NewForeignNameChanged(characterId, name).Encode(l, ctx)
	}
}

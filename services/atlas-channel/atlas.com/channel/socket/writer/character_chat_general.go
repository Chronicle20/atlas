package writer

import (
	"context"

	chatpkt "github.com/Chronicle20/atlas-packet/chat"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const CharacterChatGeneral = "CharacterChatGeneral"

func CharacterChatGeneralBody(fromCharacterId uint32, gm bool, message string, show bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewGeneralChat(fromCharacterId, gm, message, show).Encode(l, ctx)
	}
}

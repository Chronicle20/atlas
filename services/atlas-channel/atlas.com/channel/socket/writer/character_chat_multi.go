package writer

import (
	"context"

	chatpkt "github.com/Chronicle20/atlas-packet/chat"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const CharacterChatMulti = "CharacterChatMulti"

func CharacterChatMultiBody(from string, message string, mode byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return chatpkt.NewMultiChat(mode, from, message).Encode(l, ctx)
	}
}

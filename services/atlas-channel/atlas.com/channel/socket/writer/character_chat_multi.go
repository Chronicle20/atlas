package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterChatMulti = "CharacterChatMulti"

func CharacterChatMultiBody(from string, message string, mode byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(mode)
			w.WriteAsciiString(from)
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterChatGeneral = "CharacterChatGeneral"

func CharacterChatGeneralBody(fromCharacterId uint32, gm bool, message string, show bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(fromCharacterId)
			w.WriteBool(gm)
			w.WriteAsciiString(message)
			w.WriteBool(show)
			return w.Bytes()
		}
	}
}

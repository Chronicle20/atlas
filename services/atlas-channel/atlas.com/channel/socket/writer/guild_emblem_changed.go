package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	GuildEmblemChanged = "GuildEmblemChanged"
)

func ForeignGuildEmblemChangedBody(l logrus.FieldLogger) func(characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) BodyProducer {
	return func(characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteShort(logoBackground)
			w.WriteByte(logoBackgroundColor)
			w.WriteShort(logo)
			w.WriteByte(logoColor)
			return w.Bytes()
		}
	}
}

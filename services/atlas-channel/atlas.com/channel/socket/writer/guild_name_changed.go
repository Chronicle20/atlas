package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	GuildNameChanged = "GuildNameChanged"
)

func ForeignGuildNameChangedBody(_ logrus.FieldLogger) func(characterId uint32, name string) BodyProducer {
	return func(characterId uint32, name string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteAsciiString(name)
			return w.Bytes()
		}
	}
}

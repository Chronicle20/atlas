package writer

import (
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const NPCConversation = "NPCConversation"

func NPCConversationBody(l logrus.FieldLogger, t tenant.Model) func(c model.NpcConversation) BodyProducer {
	return func(c model.NpcConversation) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			c.Encode(l, t, options)(w)
			return w.Bytes()
		}
	}
}

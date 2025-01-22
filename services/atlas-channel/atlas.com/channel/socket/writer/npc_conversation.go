package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NPCConversation = "NPCConversation"

func NPCConversationBody(l logrus.FieldLogger) func(npcId uint32, talkType byte, message string, endType []byte, speaker byte) BodyProducer {
	return func(npcId uint32, talkType byte, message string, endType []byte, speaker byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(4)
			w.WriteInt(npcId)
			w.WriteByte(talkType)
			w.WriteByte(speaker)
			w.WriteAsciiString(message)
			w.WriteByteArray(endType)
			return w.Bytes()
		}
	}
}

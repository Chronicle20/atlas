package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NPCConversation = "NPCConversation"

func NPCConversationBody(l logrus.FieldLogger) func(npcId uint32, talkType byte, message string, endType []byte, speaker byte) BodyProducer {
	return func(npcId uint32, talkType byte, message string, endType []byte, speaker byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(4)        // nSpeakerTypeID
			w.WriteInt(npcId)     // nSpeakerTemplateID
			w.WriteByte(talkType) // nMsgType
			w.WriteByte(speaker)  // bParam
			w.WriteAsciiString(message)
			w.WriteByteArray(endType)
			return w.Bytes()
		}
	}
}

func NPCConversationAskNumberBody(l logrus.FieldLogger) func(npcId uint32, talkType byte, message string, def uint32, min uint32, max uint32) BodyProducer {
	return func(npcId uint32, talkType byte, message string, def uint32, min uint32, max uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(4)        // nSpeakerTypeID
			w.WriteInt(npcId)     // nSpeakerTemplateID
			w.WriteByte(talkType) // nMsgType
			w.WriteByte(0)        // bParam
			w.WriteAsciiString(message)
			w.WriteInt(def)
			w.WriteInt(min)
			w.WriteInt(max)
			return w.Bytes()
		}
	}
}

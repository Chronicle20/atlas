package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
	"strconv"
)

const (
	BuddyOperation       = "BuddyOperation"
	BuddyOperationInvite = "INVITE"
)

func BuddyInviteBody(l logrus.FieldLogger) func(actorId uint32, originatorId uint32, originatorName string) BodyProducer {
	return func(actorId uint32, originatorId uint32, originatorName string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, BuddyOperationInvite))
			w.WriteInt(originatorId)
			w.WriteAsciiString(originatorName)
			w.WriteInt(actorId)
			WritePaddedString(w, originatorName, 13)
			w.WriteByte(0) // nFlag
			w.WriteInt(0)  // nChannelID
			WritePaddedString(w, "Default Group", 17)
			w.WriteByte(0) // m_aInShop
			return w.Bytes()
		}
	}
}

func getBuddyOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var code interface{}
		if code, ok = codes[key]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, err := strconv.ParseUint(code.(string), 0, 16)
		if err != nil {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}

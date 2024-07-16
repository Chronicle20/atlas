package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NPCAction = "NPCAction"

func NPCActionAnimationBody(l logrus.FieldLogger) func(objectId uint32, unk byte, unk2 byte) BodyProducer {
	return func(objectId uint32, unk byte, unk2 byte) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteInt(objectId)
			w.WriteByte(unk)
			w.WriteByte(unk2)
			return w.Bytes()
		}
	}
}

func NPCActionMoveBody(l logrus.FieldLogger) func(objectId uint32, unk byte, unk2 byte, movement []byte) BodyProducer {
	return func(objectId uint32, unk byte, unk2 byte, movement []byte) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteInt(objectId)
			w.WriteByte(unk)
			w.WriteByte(unk2)
			w.WriteByteArray(movement)
			return w.Bytes()
		}
	}
}

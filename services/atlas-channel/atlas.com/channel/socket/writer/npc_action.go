package writer

import (
	"atlas-channel/socket/model"
	"atlas-channel/tenant"
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

func NPCActionMoveBody(l logrus.FieldLogger, tenant tenant.Model) func(objectId uint32, unk byte, unk2 byte, movePath model.Movement) BodyProducer {
	return func(objectId uint32, unk byte, unk2 byte, movePath model.Movement) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteInt(objectId)
			w.WriteByte(unk)
			w.WriteByte(unk2)
			movePath.Encode(l, tenant, options)(w)
			return w.Bytes()
		}
	}
}

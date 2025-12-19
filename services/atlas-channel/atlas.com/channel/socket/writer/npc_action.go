package writer

import (
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const NPCAction = "NPCAction"

func NPCActionAnimationBody(l logrus.FieldLogger) func(objectId uint32, unk byte, unk2 byte) BodyProducer {
	return func(objectId uint32, unk byte, unk2 byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(objectId)
			w.WriteByte(unk)
			w.WriteByte(unk2)
			return w.Bytes()
		}
	}
}

func NPCActionMoveBody(l logrus.FieldLogger, tenant tenant.Model) func(objectId uint32, unk byte, unk2 byte, movePath model.Movement) BodyProducer {
	return func(objectId uint32, unk byte, unk2 byte, movePath model.Movement) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(objectId)
			w.WriteByte(unk)
			w.WriteByte(unk2)
			movePath.Encode(l, tenant, options)(w)
			return w.Bytes()
		}
	}
}

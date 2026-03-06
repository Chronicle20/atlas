package writer

import (
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NPCAction = "NPCAction"

func NPCActionAnimationBody(objectId uint32, unk byte, unk2 byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(objectId)
			w.WriteByte(unk)
			w.WriteByte(unk2)
			return w.Bytes()
		}
	}
}

func NPCActionMoveBody(objectId uint32, unk byte, unk2 byte, movePath model.Movement) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(objectId)
			w.WriteByte(unk)
			w.WriteByte(unk2)
			w.WriteByteArray(movePath.Encode(l, ctx)(options))
			return w.Bytes()
		}
	}
}

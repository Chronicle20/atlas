package npc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NpcActionWriter = "NPCAction"

type ActionW struct {
	objectId    uint32
	unk         byte
	unk2        byte
	hasMovement bool
	movement    model.Movement
}

func NewNpcActionAnimation(objectId uint32, unk byte, unk2 byte) ActionW {
	return ActionW{objectId: objectId, unk: unk, unk2: unk2}
}

func NewNpcActionMove(objectId uint32, unk byte, unk2 byte, movement model.Movement) ActionW {
	return ActionW{objectId: objectId, unk: unk, unk2: unk2, hasMovement: true, movement: movement}
}

func (m ActionW) Operation() string { return NpcActionWriter }
func (m ActionW) String() string {
	return fmt.Sprintf("objectId [%d], hasMovement [%t]", m.objectId, m.hasMovement)
}

func (m ActionW) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.objectId)
		w.WriteByte(m.unk)
		w.WriteByte(m.unk2)
		if m.hasMovement {
			w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *ActionW) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.objectId = r.ReadUint32()
		m.unk = r.ReadByte()
		m.unk2 = r.ReadByte()
		if r.Available() > 0 {
			m.hasMovement = true
			m.movement.Decode(l, ctx)(r, options)
		}
	}
}

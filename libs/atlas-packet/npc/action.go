package npc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NPCActionHandle = "NPCActionHandle"

type Action struct {
	objectId    uint32
	unk         byte
	unk2        byte
	hasMovement bool
	movement    model.Movement
}

func (m Action) ObjectId() uint32              { return m.objectId }
func (m Action) Unk() byte                     { return m.unk }
func (m Action) Unk2() byte                    { return m.unk2 }
func (m Action) HasMovement() bool             { return m.hasMovement }
func (m Action) MovementData() model.Movement  { return m.movement }

func (m Action) Operation() string {
	return NPCActionHandle
}

func (m Action) String() string {
	if m.hasMovement {
		return fmt.Sprintf("objectId [%d] unk [%d] unk2 [%d] hasMovement [true] elements [%d]", m.objectId, m.unk, m.unk2, len(m.movement.Elements))
	}
	return fmt.Sprintf("objectId [%d] unk [%d] unk2 [%d] hasMovement [false]", m.objectId, m.unk, m.unk2)
}

func (m Action) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
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

func (m *Action) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
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

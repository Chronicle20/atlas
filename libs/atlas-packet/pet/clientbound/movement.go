package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetMovementWriter = "PetMovement"

type Movement struct {
	ownerId  uint32
	slot     int8
	movement model.Movement
}

func NewPetMovement(ownerId uint32, slot int8, movement model.Movement) Movement {
	return Movement{ownerId: ownerId, slot: slot, movement: movement}
}

func (m Movement) Operation() string { return PetMovementWriter }
func (m Movement) String() string {
	return fmt.Sprintf("ownerId [%d], slot [%d]", m.ownerId, m.slot)
}

func (m Movement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt8(m.slot)
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *Movement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.slot = r.ReadInt8()
		m.movement.Decode(l, ctx)(r, options)
	}
}

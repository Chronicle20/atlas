package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetMovementHandle = "PetMovementHandle"

type Movement struct {
	petId    uint64
	movement model.Movement
}

func (m Movement) PetId() uint64              { return m.petId }
func (m Movement) PetIdAsUint32() uint32      { return uint32(m.petId) }
func (m Movement) MovementData() model.Movement { return m.movement }

func (m Movement) Operation() string {
	return PetMovementHandle
}

func (m Movement) String() string {
	return fmt.Sprintf("petId [%d] elements [%d]", m.petId, len(m.movement.Elements))
}

func (m Movement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.petId)
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *Movement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.petId = r.ReadUint64()
		m.movement.Decode(l, ctx)(r, options)
	}
}

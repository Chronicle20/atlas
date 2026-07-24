package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const PetMovementHandle = "PetMovementHandle"

// packet-audit:fname CVecCtrlPet::EndUpdateActive
type MovementRequest struct {
	petId    uint64
	movement model.Movement
}

func (m MovementRequest) PetId() uint64                { return m.petId }
func (m MovementRequest) PetIdAsUint32() uint32        { return uint32(m.petId) }
func (m MovementRequest) MovementData() model.Movement { return m.movement }

func (m MovementRequest) Operation() string {
	return PetMovementHandle
}

func (m MovementRequest) String() string {
	return fmt.Sprintf("petId [%d] elements [%d]", m.petId, len(m.movement.Elements))
}

func (m MovementRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if hasLeadingPetId(t) {
			w.WriteLong(m.petId) // absent on GMS v48 (single-pet)
		}
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *MovementRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if hasLeadingPetId(t) {
			m.petId = r.ReadUint64() // absent on GMS v48 (single-pet)
		}
		m.movement.Decode(l, ctx)(r, options)
	}
}

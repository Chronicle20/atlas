package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const PetCommandHandle = "PetCommandHandle"

// packet-audit:fname CPet::ParseCommand
type Command struct {
	petId   uint64
	byName  bool
	command byte
}

func (m Command) PetId() uint64 {
	return m.petId
}

func (m Command) ByName() bool {
	return m.byName
}

func (m Command) Command() byte {
	return m.command
}

func (m Command) Operation() string {
	return PetCommandHandle
}

func (m Command) String() string {
	return fmt.Sprintf("petId [%d] byName [%t] command [%d]", m.petId, m.byName, m.command)
}

func (m Command) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if hasLeadingPetId(t) {
			w.WriteLong(m.petId) // absent on GMS v48 (single-pet)
		}
		w.WriteBool(m.byName)
		w.WriteByte(m.command)
		return w.Bytes()
	}
}

func (m *Command) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if hasLeadingPetId(t) {
			m.petId = r.ReadUint64() // absent on GMS v48 (single-pet)
		}
		m.byName = r.ReadBool()
		m.command = r.ReadByte()
	}
}

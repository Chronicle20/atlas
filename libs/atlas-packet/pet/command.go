package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetCommandHandle = "PetCommandHandle"

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

func (m Command) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.petId)
		w.WriteBool(m.byName)
		w.WriteByte(m.command)
		return w.Bytes()
	}
}

func (m *Command) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.petId = r.ReadUint64()
		m.byName = r.ReadBool()
		m.command = r.ReadByte()
	}
}

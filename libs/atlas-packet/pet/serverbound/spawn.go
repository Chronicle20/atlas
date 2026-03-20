package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetSpawnHandle = "PetSpawnHandle"

type Spawn struct {
	updateTime uint32
	slot       int16
	lead       bool
}

func (m Spawn) UpdateTime() uint32 {
	return m.updateTime
}

func (m Spawn) Slot() int16 {
	return m.slot
}

func (m Spawn) Lead() bool {
	return m.lead
}

func (m Spawn) Operation() string {
	return PetSpawnHandle
}

func (m Spawn) String() string {
	return fmt.Sprintf("updateTime [%d] slot [%d] lead [%t]", m.updateTime, m.slot, m.lead)
}

func (m Spawn) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.slot)
		w.WriteBool(m.lead)
		return w.Bytes()
	}
}

func (m *Spawn) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.slot = r.ReadInt16()
		m.lead = r.ReadBool()
	}
}

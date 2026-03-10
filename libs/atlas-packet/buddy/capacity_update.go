package buddy

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const BuddyCapacityUpdateWriter = "BuddyCapacityUpdate"

type CapacityUpdate struct {
	mode     byte
	capacity byte
}

func NewBuddyCapacityUpdate(mode byte, capacity byte) CapacityUpdate {
	return CapacityUpdate{mode: mode, capacity: capacity}
}

func (m CapacityUpdate) Mode() byte       { return m.mode }
func (m CapacityUpdate) Capacity() byte    { return m.capacity }
func (m CapacityUpdate) Operation() string { return BuddyCapacityUpdateWriter }

func (m CapacityUpdate) String() string {
	return fmt.Sprintf("capacity update [%d]", m.capacity)
}

func (m CapacityUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.capacity)
		return w.Bytes()
	}
}

func (m *CapacityUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.capacity = r.ReadByte()
	}
}

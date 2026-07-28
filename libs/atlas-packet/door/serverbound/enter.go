package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const EnterDoorHandle = "EnterDoorHandle"

// Enter - DoorHandler: reads ownerid (int32/uint32) then direction byte (1=town→target, 0=target→town)
type Enter struct {
	ownerId   uint32
	direction byte
}

func (m Enter) OwnerId() uint32   { return m.ownerId }
func (m Enter) Direction() byte   { return m.direction }
func (m Enter) Operation() string { return EnterDoorHandle }
func (m Enter) String() string {
	return fmt.Sprintf("Enter{ownerId=%d direction=%d}", m.ownerId, m.direction)
}

func (m Enter) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteByte(m.direction)
		return w.Bytes()
	}
}

func (m *Enter) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.direction = r.ReadByte()
	}
}

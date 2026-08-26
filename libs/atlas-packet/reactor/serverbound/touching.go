package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const TouchReactorHandle = "TouchReactorHandle"

// TouchingRequest - CReactorPool::FindTouchReactorAroundLocalUser
// packet-audit:fname CReactorPool::FindTouchReactorAroundLocalUser
type TouchingRequest struct {
	oid      uint32
	touching bool
}

func (m TouchingRequest) Oid() uint32    { return m.oid }
func (m TouchingRequest) Touching() bool { return m.touching }

func (m TouchingRequest) Operation() string {
	return TouchReactorHandle
}

func (m TouchingRequest) String() string {
	return fmt.Sprintf("oid [%d], touching [%t]", m.oid, m.touching)
}

func (m TouchingRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.oid)
		if m.touching {
			w.WriteByte(1)
		} else {
			w.WriteByte(0)
		}
		return w.Bytes()
	}
}

func (m *TouchingRequest) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.oid = r.ReadUint32()
		m.touching = r.ReadByte() == 1
	}
}

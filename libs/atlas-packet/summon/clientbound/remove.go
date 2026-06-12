package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SummonRemoveWriter = "SummonRemove"

type SummonRemove struct {
	ownerId  uint32
	oid      uint32
	animated bool
}

func NewSummonRemove(ownerId, oid uint32, animated bool) SummonRemove {
	return SummonRemove{
		ownerId:  ownerId,
		oid:      oid,
		animated: animated,
	}
}

func (m SummonRemove) OwnerId() uint32   { return m.ownerId }
func (m SummonRemove) Oid() uint32       { return m.oid }
func (m SummonRemove) Animated() bool    { return m.animated }
func (m SummonRemove) Operation() string { return SummonRemoveWriter }
func (m SummonRemove) String() string {
	return fmt.Sprintf("ownerId [%d], oid [%d], animated [%t]", m.ownerId, m.oid, m.animated)
}

func (m SummonRemove) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt(m.oid)
		if m.animated {
			w.WriteByte(4)
		} else {
			w.WriteByte(1)
		}
		return w.Bytes()
	}
}

func (m *SummonRemove) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.oid = r.ReadUint32()
		m.animated = r.ReadByte() == 4
	}
}

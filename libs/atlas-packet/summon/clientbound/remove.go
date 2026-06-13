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
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		// v95+ DELTA: oid is a v95+ addition; v83/v87 remove (sub_7A64EB) keys off
		// the dispatcher-consumed cid and reads no oid (IDB-confirmed).
		if t.IsRegion("GMS") && t.MajorAtLeast(95) {
			w.WriteInt(m.oid)
		}
		if m.animated {
			w.WriteByte(4)
		} else {
			w.WriteByte(1)
		}
		return w.Bytes()
	}
}

func (m *SummonRemove) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		if t.IsRegion("GMS") && t.MajorAtLeast(95) {
			m.oid = r.ReadUint32()
		}
		m.animated = r.ReadByte() == 4
	}
}

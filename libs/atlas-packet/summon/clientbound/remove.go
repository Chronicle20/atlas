package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
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
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		// oid: present on ALL versions. cid is read upstream by
		// CUserPool::OnUserCommonPacket; CSummonedPool::OnPacket@0x938dd7 then does
		// one Decode4 = the oid before the pool-remove (sub_7A64EB). Wire = cid + oid
		// + animated byte (the old "no oid pre-95" reading missed the upstream cid).
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
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.oid = r.ReadUint32() // present on all versions (see Encode)
		m.animated = r.ReadByte() == 4
	}
}

package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterCashItemUseHandle = "CharacterCashItemUseHandle"

// ItemUse - CUser::SendCashItemUseRequest (partial decode: common prefix only).
//
// The common prefix carries update_time as a leading int32 on all versions from
// GMS v87 onward (and JMS v185), where CWvsContext::SendConsumeCashItemUseRequest
// encodes get_update_time() in the header BEFORE the per-type sub-body:
//   - gms_v87 @0xa9fef9: Encode4(update_time); Encode2(nPOS); Encode4(nItemID); ...
//   - jms_v185 @0xaef2f5: Encode4(update_time); Encode2(nPOS); Encode4(nItemID); ...
//   - gms_v95 @0x9eb3e0: same (header-first) — the original >=95 gate.
//
// The two oldest versions (gms_v83 @0xa0a63f, gms_v84 by byte-identity) instead
// append update_time as a TRAILING int32 in the send tail, so they omit it from
// this header. Hence the layout switch is MajorVersion() >= 87, not >= 95.
type ItemUse struct {
	updateTime uint32
	source     int16
	itemId     uint32
}

func (m ItemUse) UpdateTime() uint32 { return m.updateTime }
func (m ItemUse) Source() int16      { return m.source }
func (m ItemUse) ItemId() uint32     { return m.itemId }

func (m ItemUse) Operation() string {
	return CharacterCashItemUseHandle
}

func (m ItemUse) String() string {
	return fmt.Sprintf("updateTime [%d], source [%d], itemId [%d]", m.updateTime, m.source, m.itemId)
}

func (m ItemUse) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.MajorVersion() >= 87 {
			w.WriteInt(m.updateTime)
		}
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *ItemUse) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.MajorVersion() >= 87 {
			m.updateTime = r.ReadUint32()
		}
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}

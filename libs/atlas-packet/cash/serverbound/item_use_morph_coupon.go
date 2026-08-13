package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest
//
// ItemUseMorphCoupon is the USE_CASH_ITEM sub-body for a transformation
// (morph) coupon, item classification 530 — cash-slot type 40 on GMS < 95 and
// 41 on GMS >= 95 (see GetCashSlotItemType in atlas-channel).
//
// The sub-body is EMPTY. IDA-verified on GMS v83 (MapleStory_dump.exe,
// CWvsContext::SendConsumeCashItemUseRequest @0xa0a63f): the case-40 arm spans
// 0xa0caf0-0xa0cb37 and contains no Encode* call at all. It runs three
// client-side predicates — the first, sub_A0ECCD @0xa0eccd, is literally
// `itemId / 10000 == 530` — then calls play_item_sound(nItemID, 0x29) @0xa0cb30
// and jumps to the shared send tail. The tail is what appends the trailing
// Encode4(get_update_time()) @0xa0ea5c on the versions that trail it.
//
// So the only thing this codec carries is that trailing updateTime, and only on
// the versions where the common ItemUse header did not already read it: GMS
// <= v84 trail, GMS v87+ and JMS lead (cashsb.UpdateTimeFirst). Byte-identical
// in behaviour to ItemUsePetConsumable, kept as its own type because the
// package convention is one struct per client arm and a future divergence on
// either arm must not force an unpick of the other.
type ItemUseMorphCoupon struct {
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseMorphCoupon(updateTimeFirst bool) *ItemUseMorphCoupon {
	return &ItemUseMorphCoupon{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseMorphCoupon) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseMorphCoupon) Operation() string { return "ItemUseMorphCoupon" }

func (m ItemUseMorphCoupon) String() string {
	return fmt.Sprintf("updateTime [%d]", m.updateTime)
}

func (m ItemUseMorphCoupon) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseMorphCoupon) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}

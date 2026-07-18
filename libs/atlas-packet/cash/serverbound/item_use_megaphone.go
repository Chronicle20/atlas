package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseMegaphone is the USE_CASH_ITEM sub-body for the basic Megaphone
// (5071xxx, cash-slot type 12 within classification 507).
// Cosmic-derived (UseCashItemHandler case 1); per-version IDA verification in
// task-123 phases 19-20, legacy phase 1, legacy TV/item/triple gap-fill pass.
//
// CORRECTION (legacy TV/item/triple gap-fill pass): the earlier
// megaphoneHasUpdateTime(GMS<83)=false gate was WRONG. Basic/super Megaphone
// (cases 12/13) only LOOK like they omit the trailing update_time when the
// trace stops at the case body's own cleanup — every legacy build actually
// falls through (via a `jz`/`jnz` on an unrelated "attached commodity"
// pointer, normally nil) into a SHARED rate-check tail that, on success,
// does `call SetExclRequestSent (== a GetTickCount-style read of
// g_CWvsApp+0x18); push eax; call Encode4; call SendPacket`. IDA-verified
// end-to-end on all four legacy builds: gms_v48 case-34 tail @0x711d96,
// gms_v61 (same architecture as v72/79 below), gms_v72 case-33 tail
// @0x90911a (reached from case 12/13's body via `jz loc_905294` @0x9055f5),
// gms_v79 (byte-identical structure to v72). update_time is therefore
// present (trailing) on EVERY GMS build with updateTimeFirst=false — v83/84
// AND v48/61/72/79 alike — never a GMS<83-only-absent field. This matches
// ItemUseItemMegaphone/ItemUseTripleMegaphone/ItemUseMapleTV, none of which
// ever had a extra hasUpdateTime gate.
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest
type ItemUseMegaphone struct {
	message         string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseMegaphone(updateTimeFirst bool) *ItemUseMegaphone {
	return &ItemUseMegaphone{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseMegaphone) Message() string    { return m.message }
func (m ItemUseMegaphone) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseMegaphone) Operation() string { return "ItemUseMegaphone" }

func (m ItemUseMegaphone) String() string {
	return fmt.Sprintf("message [%s] updateTime [%d]", m.message, m.updateTime)
}

func (m ItemUseMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}

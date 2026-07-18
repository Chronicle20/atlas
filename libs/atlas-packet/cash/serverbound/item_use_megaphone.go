package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// megaphoneHasUpdateTime reports whether the basic/super Megaphone sub-body
// carries an update_time field AT ALL (leading in the outer ItemUse header
// from v87, trailing in the sub-body on v83/84 — see ItemUse's doc comment).
// Legacy GMS (v48/61/72/79, MajorVersion<83) carry NEITHER: the client's
// CWvsContext::SendConsumeCashItemUseRequest cases 12/13 send
// EncodeStr(message) [+Encode1(whisper) for the super-arm] with NO trailing
// Encode4 anywhere, and the shared outer header (COutPacket ctor, Encode2
// (slot), Encode4(itemId)) has no update_time either. IDA-verified 4/4 legacy
// anchors (task-123 legacy phase 1): gms_v48 @0x70e800 (case block start
// loc_70E543/jumptable cases 12,13), gms_v61 @0x832ddc (loc_832B08 cases
// 12,13), gms_v72 @0x9055ad (loc_9052D3 cases 12,13,15), gms_v79 @0x956919
// (loc_95663B cases 12,13,15) — each traced straight through EncodeStr(
// message)+[Encode1(whisper)] to case cleanup, zero Encode4 calls. update_time
// is therefore a GMS>=83 (+JMS) wire concept, absent entirely below v83.
func megaphoneHasUpdateTime(t tenant.Model) bool {
	return !(t.Region() == "GMS" && t.MajorVersion() < 83)
}

// ItemUseMegaphone is the USE_CASH_ITEM sub-body for the basic Megaphone
// (5071xxx, cash-slot type 12 within classification 507).
// Cosmic-derived (UseCashItemHandler case 1); per-version IDA verification in task-123 phases 19-20, legacy phase 1.
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

func (m ItemUseMegaphone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst && megaphoneHasUpdateTime(t) {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseMegaphone) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst && megaphoneHasUpdateTime(t) {
			m.updateTime = r.ReadUint32()
		}
	}
}

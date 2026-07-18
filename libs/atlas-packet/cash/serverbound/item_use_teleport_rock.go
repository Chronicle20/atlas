package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseTeleportRock - the teleport-rock branch of
// CWvsContext::SendConsumeCashItemUseRequest (case 22 — jumptable 009EB50A on
// GMS_v95.0_U_DEVM.exe), after the common ItemUse prefix: the shared
// RunMapTransferItem target payload, then a trailing int updateTime ONLY on
// MajorVersion()<87 (v83/v84 — mirrors CashItemUsePointReset's
// updateTimeFirst gate).
//
// task-124 v95 verify pass (live GMS_v95.0_U_DEVM.exe, port 13341):
// CWvsContext::SendConsumeCashItemUseRequest @0x9eb3e0 encodes update_time
// FIRST in the common header prologue (Encode4 @0x9eb4b7, BEFORE
// Encode2(nPOS)/Encode4(nItemID) and the switch) — already the parent
// ItemUse's leading updateTime, per updateTimeFirst := MajorVersion()>=87 in
// character_cash_item_use.go. Case 22 ($LN84_16 @0x9ee059) computes
// bCanTransferContinent = (nItemID / 5040 != 5040 via 0x10624DD3 magic-multiply)
// and calls RunMapTransferItem(this, &oPacket, flag) @0x9ee080 — the SAME
// helper USE_TELEPORT_ROCK calls (0x9e11c0) — then falls straight to the
// shared send tail ($LN232_14 @0x9f063c: CanSendExclRequest + SendPacket) with
// NO further Encode4 anywhere in that path. So for v87+ (incl. v95/jms) the
// sub-body on the wire is EXACTLY the target payload — nothing trailing.
// This CORRECTS the original task-124 design hypothesis (§1 Q1, which assumed
// a v95 trailing updateTime at the case-22 tail) — the true v95 sub-body has
// no trailing field at all; updateTimeFirst was already threaded through by
// the caller but this codec ignored it until this pass wired the gate below.
type ItemUseTeleportRock struct {
	updateTimeFirst bool
	target          teleportrock.Target
	updateTime      uint32
}

func NewItemUseTeleportRock(updateTimeFirst bool) *ItemUseTeleportRock {
	return &ItemUseTeleportRock{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseTeleportRock) Target() teleportrock.Target { return m.target }
func (m ItemUseTeleportRock) UpdateTime() uint32          { return m.updateTime }

func (m ItemUseTeleportRock) String() string {
	return fmt.Sprintf("ItemUseTeleportRock{target=%s updateTime=%d}", m.target.String(), m.updateTime)
}

func (m ItemUseTeleportRock) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		m.target.Encode(w)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseTeleportRock) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.target.Decode(l, !m.updateTimeFirst)(r)
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}

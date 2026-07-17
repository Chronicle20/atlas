package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const IncubatorResultWriter = "IncubatorResult"

// IncubatorResult is CWvsContext::OnIncubatorResult. itemId <= 0 renders the
// client's "inventory is full, try again later" dialog.
//
// v83 (0xa28298) / v84 (0xa73a5b) / v87 (0xabff10) / JMS (0xb0f30b): int
// itemId, short count (6 bytes) — live IDA re-verified for all four; none of
// them read anything past the count field.
// v95 (0xa00380) only: those two fields plus gachaponItemId (the sacrificed
// Pigmy Egg, used by the client to pick the region success NPC via
// GetGachaponSucessNpc) and a trailing bonus pair (bonusItemId, bonusCount)
// — Atlas rolls a single reward so the bonus pair is always zero and the
// client skips the bonus branch.
type IncubatorResult struct {
	itemId         uint32
	count          uint16
	gachaponItemId uint32
}

// NewIncubatorResult constructs an IncubatorResult. itemId <= 0 signals
// failure ("inventory is full, try again later") to the client.
// gachaponItemId is the sacrificed Pigmy Egg id; the v95 client uses it to
// pick the region success NPC (GetGachaponSucessNpc). Pass 0 on
// failure/older versions.
func NewIncubatorResult(itemId uint32, count uint16, gachaponItemId uint32) IncubatorResult {
	return IncubatorResult{itemId: itemId, count: count, gachaponItemId: gachaponItemId}
}

func (m IncubatorResult) ItemId() uint32         { return m.itemId }
func (m IncubatorResult) Count() uint16          { return m.count }
func (m IncubatorResult) GachaponItemId() uint32 { return m.gachaponItemId }
func (m IncubatorResult) Operation() string      { return IncubatorResultWriter }

// Encode encodes the OnIncubatorResult body (no opcode — config-driven at
// runtime). The v95-only extended tail is version-switched here, matching the
// model/asset.go idiom, rather than via a constructor flag.
func (m IncubatorResult) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.itemId)
		w.WriteShort(m.count)
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			// v95 reads gachaponItemID (the sacrificed egg → region NPC) then a
			// bonus pair. Atlas rolls one reward, so the bonus pair stays zero.
			w.WriteInt(m.gachaponItemId)
			w.WriteInt(0)
			w.WriteInt(0)
		}
		return w.Bytes()
	}
}

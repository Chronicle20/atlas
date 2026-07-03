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
// v83 (0xa28298) / v84: int itemId, short count (6 bytes).
// v87 (0xa00380) / v95 / JMS: those two fields plus three trailing zero ints
// (gachaponItemId, bonusItemId, bonusCount) — Atlas rolls a single reward so
// the bonus tail is always zero and the client skips the bonus branch.
type IncubatorResult struct {
	itemId uint32
	count  uint16
}

// NewIncubatorResult constructs an IncubatorResult packet. itemId <= 0 signals
// failure ("inventory is full, try again later") to the client.
func NewIncubatorResult(itemId uint32, count uint16) IncubatorResult {
	return IncubatorResult{itemId: itemId, count: count}
}

func (m IncubatorResult) ItemId() uint32    { return m.itemId }
func (m IncubatorResult) Count() uint16     { return m.count }
func (m IncubatorResult) Operation() string { return IncubatorResultWriter }

// Encode encodes the OnIncubatorResult body (no opcode — config-driven at
// runtime). The v87+/JMS extended tail is version-switched here, matching the
// model/asset.go idiom, rather than via a constructor flag.
func (m IncubatorResult) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.itemId)
		w.WriteShort(m.count)
		if (t.Region() == "GMS" && t.MajorVersion() >= 87) || t.Region() == "JMS" {
			// Atlas rolls a single reward; the gachapon/bonus tail is unused.
			w.WriteInt(0)
			w.WriteInt(0)
			w.WriteInt(0)
		}
		return w.Bytes()
	}
}

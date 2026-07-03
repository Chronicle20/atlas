package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestItemUseVegaScrollRoundTrip(t *testing.T) {
	input := NewItemUseVegaScroll(1, 5, 2, 7, 1, 305419896)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ItemUseVegaScroll{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestItemUseVegaScrollBytes locks the six-int32 little-endian wire shape:
// equipTab(1) equipSlot(5) scrollTab(2) scrollSlot(7) flag(1)
// updateTime(0x12345678) — 24 bytes, version-independent (no gate in codec).
//
// The 6-int32 body was IDA-verified at the CUIVega send site on all four live
// clients (task-130 Task 4): the body is assembled and sent by the CUIVega
// dialog (NOT CWvsContext::SendConsumeCashItemUseRequest directly), each
// writing Encode4(equipTab=1), Encode4(equipSlot), Encode4(scrollTab=2),
// Encode4(scrollSlot), Encode4(flag=1), Encode4(get_update_time()):
//   - gms_v83  sub_82CBE2  LABEL_28  @0x82cbe2
//   - gms_v87  sub_890CA2  LABEL_28  @0x890ca2
//   - gms_v95  CUIVega::OnButtonClicked LABEL_12 @0x7bf4a0
//   - jms_v185 sub_8B7CF1  @0x8b7cf1
// No packet-audit:verify marker is carried here: the serverbound cell is the
// SHARED USE_CASH_ITEM opcode whose audit report is owned by the shared
// cash-item-use pass (task-126, not yet landed). Adding a marker without that
// shared report would register as an orphan. The vega arm's evidence splices
// into that shared audit when it lands (brief Step 2 coordination note).
func TestItemUseVegaScrollBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewItemUseVegaScroll(1, 5, 2, 7, 1, 0x12345678)
	want := "01000000" + "05000000" + "02000000" + "07000000" + "01000000" + "78563412"
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := hex.EncodeToString(input.Encode(l, ctx)(nil))
			if got != want {
				t.Errorf("bytes: got %s, want %s", got, want)
			}
		})
	}
}

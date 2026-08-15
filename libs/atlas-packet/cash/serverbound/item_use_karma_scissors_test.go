package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseKarmaScissorsRoundTrip(t *testing.T) {
	for _, first := range []bool{true, false} {
		for _, v := range pt.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				input := ItemUseKarmaScissors{inventoryType: 1, slot: 3, updateTime: 1000, updateTimeFirst: first}
				output := *NewItemUseKarmaScissors(first)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.InventoryType() != input.InventoryType() {
					t.Errorf("inventoryType = %d, want %d", output.InventoryType(), input.InventoryType())
				}
				if output.Slot() != input.Slot() {
					t.Errorf("slot = %d, want %d", output.Slot(), input.Slot())
				}
				if !first && output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime = %d, want %d", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}

// v83 golden bytes (CUIKarmaDlg::_SendConsumeCashItemUseRequest @0x830FB5, which
// TRAILS update_time): int nTargetTI (1) + int nTargetPOS (3) + int update_time (1000).
func TestItemUseKarmaScissorsV83Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseKarmaScissors{inventoryType: 1, slot: 3, updateTime: 1000, updateTimeFirst: false}
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0x01, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0xE8, 0x03, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

// v95 golden bytes (@0x7D7EF0, which LEADS update_time in the common ItemUse
// header): the sub-body is int nTargetTI (1) + int nTargetPOS (3) and nothing else.
func TestItemUseKarmaScissorsV95Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseKarmaScissors{inventoryType: 1, slot: 3, updateTime: 1000, updateTimeFirst: true}
	got := m.Encode(l, pt.CreateContext("GMS", 95, 0))(nil)
	want := []byte{0x01, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

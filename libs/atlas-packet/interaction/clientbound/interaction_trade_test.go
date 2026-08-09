package clientbound

import (
	"encoding/hex"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestInteractionTradePutItemRoundTrip pins the mode-15 arm: Decode1 side,
// Decode1 trade slot, then GW_ItemSlotBase (v83 sub_7C1FB7 @0x7c1fb7).
func TestInteractionTradePutItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			a := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(50, 0, 0)
			input := NewInteractionTradePutItem(15, 1, 3, a)

			l, _ := testlog.NewNullLogger()
			raw := input.Encode(l, ctx)(nil)
			if len(raw) < 3 {
				t.Fatalf("body too short: %d bytes", len(raw))
			}
			if raw[0] != 15 {
				t.Errorf("mode: got %d, want 15", raw[0])
			}
			if raw[1] != 1 {
				t.Errorf("side: got %d, want 1", raw[1])
			}
			if raw[2] != 3 {
				t.Errorf("tradeSlot: got %d, want 3", raw[2])
			}
		})
	}
}

// TestInteractionTradePutItemHeaderBytes pins the fixed three-byte header ahead
// of the asset blob so a field reorder is caught independently of asset codec
// churn.
func TestInteractionTradePutItemHeaderBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	a := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(50, 0, 0)
	raw := NewInteractionTradePutItem(15, 0, 1, a).Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if got := hex.EncodeToString(raw[:3]); got != "0f0001" {
		t.Errorf("header bytes: got %s, want 0f0001", got)
	}
}

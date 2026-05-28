package clientbound

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func testAsset() model.Asset {
	return model.NewAsset(true, 0, 2000000, time.Time{}).
		SetStackableInfo(5, 0, 0)
}

// etcAsset builds an asset in the ETC tab so the per-tab segmentation in
// Show.Encode can be exercised across multiple tabs. 4000000 is an ETC item.
func etcAsset() model.Asset {
	return model.NewAsset(true, 0, 4000000, time.Time{}).SetStackableInfo(3, 0, 0)
}

// TestStorageShowSegmentation verifies per-tab segmentation: with the currency
// bit and two tab bits set, the body is meso + one count+items block per set tab
// (USE then ETC), with NO leading or trailing padding shorts/bytes. Read order
// confirmed against IDA v83 CTrunkDlg::SetGetItems@0x7c5dfd and v95
// CTrunkDlg::SetGetItems@0x76a390 (identical per-tab loop over bits 4/8/16/32/64,
// meso gated on flag&2). Wire bug present in both versions; fix unconditional.
func TestStorageShowSegmentation(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// flags: currency(2) | use(8) | etc(32) = 42
			assets := []model.Asset{etcAsset(), testAsset()} // ETC first to verify reorder
			input := NewStorageShow(22, 9200000, 16, 2|8|32, 50000, assets)
			b := input.Encode(l, ctx)(nil)

			// Header: mode(1) npc(4) slots(1) flags(8) = 14 bytes.
			if b[0] != 22 {
				t.Fatalf("mode: got %d", b[0])
			}
			// meso present (currency bit set) at offset 14..18
			meso := uint32(b[14]) | uint32(b[15])<<8 | uint32(b[16])<<16 | uint32(b[17])<<24
			if meso != 50000 {
				t.Errorf("meso: got %d, want 50000", meso)
			}
			// next byte is the USE-tab count (1)
			if b[18] != 1 {
				t.Errorf("use count: got %d, want 1", b[18])
			}

			// Round-trip and confirm both assets returned, no leftover bytes.
			output := Show{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Assets()) != 2 {
				t.Fatalf("assets: got %d, want 2", len(output.Assets()))
			}
		})
	}
}

// TestStorageShowMesoGate verifies meso is omitted when the currency bit is
// clear (flag&2 == 0), matching the conditional Decode4 in SetGetItems.
func TestStorageShowMesoGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	// flags: use(8) only, no currency bit
	input := NewStorageShow(22, 9200000, 16, 8, 50000, []model.Asset{testAsset()})
	b := input.Encode(l, ctx)(nil)
	// Header 14 bytes, then immediately the USE count byte (1) — no meso int.
	if len(b) < 15 {
		t.Fatalf("encoded too short: %d", len(b))
	}
	if b[14] != 1 {
		t.Errorf("expected USE count byte at offset 14 (no meso), got %d", b[14])
	}
	output := Show{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if len(output.Assets()) != 1 {
		t.Errorf("assets: got %d, want 1", len(output.Assets()))
	}
}

func TestStorageShowRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			assets := []model.Asset{testAsset()}
			input := NewStorageShow(22, 9200000, 16, 126, 50000, assets)
			output := Show{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.NpcId() != input.NpcId() {
				t.Errorf("npcId: got %v, want %v", output.NpcId(), input.NpcId())
			}
			if output.Slots() != input.Slots() {
				t.Errorf("slots: got %v, want %v", output.Slots(), input.Slots())
			}
			if output.Flags() != input.Flags() {
				t.Errorf("flags: got %v, want %v", output.Flags(), input.Flags())
			}
			if output.Meso() != input.Meso() {
				t.Errorf("meso: got %v, want %v", output.Meso(), input.Meso())
			}
			if len(output.Assets()) != 1 {
				t.Fatalf("assets: got %d, want 1", len(output.Assets()))
			}
			if output.Assets()[0].TemplateId() != 2000000 {
				t.Errorf("templateId: got %v, want 2000000", output.Assets()[0].TemplateId())
			}
		})
	}
}

func TestStorageShowEmptyRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewStorageShow(22, 9200000, 16, 126, 0, nil)
	output := Show{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if len(output.Assets()) != 0 {
		t.Errorf("assets: got %d, want 0", len(output.Assets()))
	}
}

func TestStorageUpdateAssetsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			assets := []model.Asset{testAsset(), testAsset()}
			input := NewStorageUpdateAssets(9, 16, 8, assets)
			output := UpdateAssets{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Slots() != input.Slots() {
				t.Errorf("slots: got %v, want %v", output.Slots(), input.Slots())
			}
			if output.Flags() != input.Flags() {
				t.Errorf("flags: got %v, want %v", output.Flags(), input.Flags())
			}
			if len(output.Assets()) != 2 {
				t.Fatalf("assets: got %d, want 2", len(output.Assets()))
			}
		})
	}
}

package storage

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-packet/model"
	pt "github.com/Chronicle20/atlas-packet/test"
)

func testAsset() model.Asset {
	return model.NewAsset(true, 0, 2000000, time.Time{}).
		SetStackableInfo(5, 0, 0)
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

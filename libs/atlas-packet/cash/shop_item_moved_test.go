package cash

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-packet/model"
	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCashItemMovedToCashInventoryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCashItemMovedToCashInventory(0x50, testItem())
			output := CashItemMovedToCashInventory{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Item().CashId != input.Item().CashId {
				t.Errorf("cashId: got %v, want %v", output.Item().CashId, input.Item().CashId)
			}
			if output.Item().TemplateId != input.Item().TemplateId {
				t.Errorf("templateId: got %v, want %v", output.Item().TemplateId, input.Item().TemplateId)
			}
		})
	}
}

func TestCashItemMovedToInventoryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			asset := model.NewAsset(true, 0, 2000000, time.Time{}).
				SetStackableInfo(5, 0, 0)
			input := NewCashItemMovedToInventory(0x51, 3, asset)
			output := CashItemMovedToInventory{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Asset().TemplateId() != input.Asset().TemplateId() {
				t.Errorf("templateId: got %v, want %v", output.Asset().TemplateId(), input.Asset().TemplateId())
			}
		})
	}
}

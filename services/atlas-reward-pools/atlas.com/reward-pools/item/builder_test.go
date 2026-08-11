package item_test

import (
	"atlas-reward-pools/gachapon"
	"atlas-reward-pools/item"
	"atlas-reward-pools/test"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestBuilderValidation(t *testing.T) {
	tenantId := test.TestTenantId

	tests := []struct {
		name    string
		tier    string
		wantErr bool
	}{
		{name: "valid tier common", tier: "common", wantErr: false},
		{name: "valid tier uncommon", tier: "uncommon", wantErr: false},
		{name: "valid tier rare", tier: "rare", wantErr: false},
		{name: "invalid tier", tier: "invalid", wantErr: true},
		{name: "empty tier", tier: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := item.NewBuilder(tenantId, 0).
				SetGachaponId("gachapon-1").
				SetItemId(1000).
				SetQuantity(1).
				SetTier(tt.tier).
				Build()

			if tt.wantErr && err == nil {
				t.Errorf("Expected error for tier %q, got nil", tt.tier)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error for tier %q, got: %v", tt.tier, err)
			}
		})
	}
}

func TestBuilderWeight(t *testing.T) {
	tenantId := test.TestTenantId

	tests := []struct {
		name       string
		setWeight  bool
		weight     uint32
		wantWeight uint32
	}{
		{
			name:       "defaults to 0 when SetWeight is never called",
			setWeight:  false,
			wantWeight: 0,
		},
		{
			name:       "SetWeight overrides the default",
			setWeight:  true,
			weight:     50,
			wantWeight: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := item.NewBuilder(tenantId, 0).
				SetGachaponId("gachapon-1").
				SetItemId(1000).
				SetQuantity(1).
				SetTier("common")
			if tt.setWeight {
				b = b.SetWeight(tt.weight)
			}
			m, err := b.Build()
			if err != nil {
				t.Fatalf("Build() returned error: %v", err)
			}
			if m.Weight() != tt.wantWeight {
				t.Errorf("Expected Weight() = %d, got %d", tt.wantWeight, m.Weight())
			}
		})
	}
}

func TestBuilderCashSurpriseRequiresCommodityId(t *testing.T) {
	_, err := item.NewBuilder(uuid.New(), 1).
		SetGachaponId("5222000").
		SetKind(gachapon.KindCashSurprise).
		SetItemId(5222001).
		SetQuantity(1).
		SetTier("common").
		SetWeight(10).
		Build()
	if !errors.Is(err, item.ErrCommodityIdRequired) {
		t.Fatalf("err = %v, want ErrCommodityIdRequired — a cash-surprise entry without a commodity cannot be granted", err)
	}
}

func TestBuilderCashSurpriseAcceptsCommodityId(t *testing.T) {
	m, err := item.NewBuilder(uuid.New(), 1).
		SetGachaponId("5222000").
		SetKind(gachapon.KindCashSurprise).
		SetItemId(5222001).
		SetQuantity(1).
		SetTier("common").
		SetWeight(10).
		SetCommodityId(40000).
		Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if m.CommodityId() != 40000 {
		t.Fatalf("commodityId = %d, want 40000", m.CommodityId())
	}
}

// Existing kinds must be untouched: a gachapon or incubator entry with no
// commodity id still builds, and reads 0.
func TestBuilderOtherKindsDoNotRequireCommodityId(t *testing.T) {
	for _, kind := range []string{gachapon.KindGachapon, gachapon.KindIncubator, ""} {
		m, err := item.NewBuilder(uuid.New(), 1).
			SetGachaponId("9000000").
			SetKind(kind).
			SetItemId(2000000).
			SetQuantity(1).
			SetTier("common").
			Build()
		if err != nil {
			t.Fatalf("kind %q: build failed: %v", kind, err)
		}
		if m.CommodityId() != 0 {
			t.Fatalf("kind %q: commodityId = %d, want 0", kind, m.CommodityId())
		}
	}
}

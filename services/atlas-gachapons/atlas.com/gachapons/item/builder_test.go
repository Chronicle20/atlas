package item_test

import (
	"atlas-gachapons/item"
	"atlas-gachapons/test"
	"testing"
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

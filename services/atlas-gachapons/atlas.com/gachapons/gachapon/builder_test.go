package gachapon_test

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/test"
	"testing"
)

func TestBuilderKind(t *testing.T) {
	tenantId := test.TestTenantId

	tests := []struct {
		name     string
		setKind  string // "" means SetKind is never called
		callSet  bool
		wantErr  bool
		wantKind string
	}{
		{
			name:     "defaults to gachapon when SetKind is never called",
			callSet:  false,
			wantErr:  false,
			wantKind: gachapon.KindGachapon,
		},
		{
			name:     "SetKind overrides the default with incubator",
			setKind:  gachapon.KindIncubator,
			callSet:  true,
			wantErr:  false,
			wantKind: gachapon.KindIncubator,
		},
		{
			name:     "SetKind accepts gachapon explicitly",
			setKind:  gachapon.KindGachapon,
			callSet:  true,
			wantErr:  false,
			wantKind: gachapon.KindGachapon,
		},
		{
			name:    "SetKind with an invalid kind is rejected by Build",
			setKind: "not-a-real-kind",
			callSet: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := gachapon.NewBuilder(tenantId, "henesys").
				SetName("Henesys").
				SetNpcIds([]uint32{9100100}).
				SetCommonWeight(70).
				SetUncommonWeight(25).
				SetRareWeight(5)
			if tt.callSet {
				b = b.SetKind(tt.setKind)
			}
			m, err := b.Build()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Build() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Build() returned error: %v", err)
			}
			if m.Kind() != tt.wantKind {
				t.Errorf("Expected Kind() = %q, got %q", tt.wantKind, m.Kind())
			}
		})
	}
}

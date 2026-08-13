package gachapon_test

import (
	"atlas-reward-pools/gachapon"
	"atlas-reward-pools/test"
	"testing"

	"github.com/google/uuid"
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

func TestBuilderAcceptsCashSurpriseKind(t *testing.T) {
	m, err := gachapon.NewBuilder(uuid.New(), "5222000").
		SetName("Cash Shop Surprise").
		SetKind(gachapon.KindCashSurprise).
		Build()
	if err != nil {
		t.Fatalf("cash-surprise kind rejected: %v", err)
	}
	if m.Kind() != gachapon.KindCashSurprise {
		t.Fatalf("kind = %q, want %q", m.Kind(), gachapon.KindCashSurprise)
	}
}

func TestBuilderStillRejectsUnknownKind(t *testing.T) {
	_, err := gachapon.NewBuilder(uuid.New(), "1").SetKind("mystery-box").Build()
	if err == nil {
		t.Fatal("unknown kind must be rejected — the union stays closed")
	}
}

func TestDefaultKindUnchanged(t *testing.T) {
	m, err := gachapon.NewBuilder(uuid.New(), "9000000").Build()
	if err != nil {
		t.Fatalf("default build failed: %v", err)
	}
	if m.Kind() != gachapon.KindGachapon {
		t.Fatalf("DefaultKind regressed to %q — existing rows read this value", m.Kind())
	}
}

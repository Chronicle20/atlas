package parcel

import "testing"

func TestFee(t *testing.T) {
	tests := []struct {
		meso     uint32
		expected uint32
	}{
		{0, 0},
		{99_999, 0},
		{100_000, 800},
		{999_999, 7_999},
		{1_000_000, 18_000},
		{4_999_999, 89_999},
		{5_000_000, 150_000},
		{9_999_999, 299_999},
		{10_000_000, 400_000},
		{24_999_999, 999_999},
		{25_000_000, 1_250_000},
		{99_999_999, 4_999_999},
		{100_000_000, 6_000_000},
	}

	for _, tt := range tests {
		if got := Fee(tt.meso); got != tt.expected {
			t.Errorf("Fee(%d) = %d, want %d", tt.meso, got, tt.expected)
		}
	}
}

func TestTotalCost(t *testing.T) {
	tests := []struct {
		name     string
		meso     uint32
		quick    bool
		expected uint64
		ok       bool
	}{
		{"zero, npc arm", 0, false, 5_000, true},
		{"zero, quick", 0, true, 0, true},
		{"tier one, npc arm", 100_000, false, 105_800, true},
		{"tier one, quick", 100_000, true, 100_800, true},
		{"max parcel, npc arm", 100_000_000, false, 106_005_000, true},
		{"overflows uint32", 4_294_000_000, false, 0, false},
		{"just below the uint32 boundary", 4_051_855_938, true, 4_294_967_294, true},
		{"exactly at the uint32 boundary", 4_051_855_939, true, 4_294_967_295, true},
		{"just above the uint32 boundary", 4_051_855_940, true, 4_294_967_296, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, ok := TotalCost(tt.meso, tt.quick)
			if ok != tt.ok {
				t.Fatalf("TotalCost(%d, %v) ok = %v, want %v", tt.meso, tt.quick, ok, tt.ok)
			}
			if tt.ok && total != tt.expected {
				t.Errorf("TotalCost(%d, %v) = %d, want %d", tt.meso, tt.quick, total, tt.expected)
			}
		})
	}
}

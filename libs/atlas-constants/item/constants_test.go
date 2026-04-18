package item

import "testing"

func TestIsArrow(t *testing.T) {
	// Classification 206 — arrows for bows (e.g. 2060000) and bolts for crossbows (e.g. 2061000).
	if !IsArrow(Id(2060000)) {
		t.Fatalf("expected 2060000 to be classified as arrow")
	}
	if !IsArrow(Id(2061000)) {
		t.Fatalf("expected 2061000 (crossbow bolt) to be classified as arrow")
	}
	if IsArrow(Id(2070000)) {
		t.Fatalf("throwing star 2070000 should not be arrow")
	}
	if IsArrow(Id(2330000)) {
		t.Fatalf("bullet 2330000 should not be arrow")
	}
	if IsArrow(Id(2000000)) {
		t.Fatalf("generic consumable 2000000 should not be arrow")
	}
}

func TestIsRechargeable(t *testing.T) {
	if !IsRechargeable(Id(2070000)) {
		t.Fatalf("throwing star 2070000 should be rechargeable")
	}
	if !IsRechargeable(Id(2330000)) {
		t.Fatalf("bullet 2330000 should be rechargeable")
	}
	if IsRechargeable(Id(2060000)) {
		t.Fatalf("arrow 2060000 should not be rechargeable")
	}
	if IsRechargeable(Id(2000000)) {
		t.Fatalf("generic consumable 2000000 should not be rechargeable")
	}
}

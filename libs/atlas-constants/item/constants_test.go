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

func TestIsTwoHanded(t *testing.T) {
	cases := []struct {
		name string
		id   Id
		want bool
	}{
		{"one-handed sword 130xxxx", Id(1302000), false},
		{"dagger 133xxxx", Id(1332000), false},
		{"wand 137xxxx", Id(1372000), false},
		{"two-handed sword 140xxxx", Id(1402000), true},
		{"polearm 144xxxx", Id(1442000), true},
		{"bow 145xxxx", Id(1452000), true},
		{"crossbow 146xxxx", Id(1462000), true},
		{"claw 147xxxx (one-handed)", Id(1472000), false},
		{"knuckle 148xxxx", Id(1482000), true},
		{"gun 149xxxx", Id(1492000), true},
		{"non-weapon hat 100xxxx", Id(1002000), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTwoHanded(tc.id); got != tc.want {
				t.Fatalf("IsTwoHanded(%d) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

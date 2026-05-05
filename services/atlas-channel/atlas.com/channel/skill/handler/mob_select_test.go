package handler

import (
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

func mkPoint(x, y int16) point.Model {
	return point.NewModel(point.X(x), point.Y(y))
}

func TestBoundingBox_FacingRight_SymmetricRect(t *testing.T) {
	lt := mkPoint(-200, -100)
	rb := mkPoint(200, 100)
	x1, y1, x2, y2 := calculateBoundingBox(0, 0, false, lt, rb)
	if x1 != -200 || y1 != -100 || x2 != 200 || y2 != 100 {
		t.Fatalf("got (%d,%d,%d,%d), want (-200,-100,200,100)", x1, y1, x2, y2)
	}
}

func TestBoundingBox_FacingLeft_SymmetricRect(t *testing.T) {
	lt := mkPoint(-200, -100)
	rb := mkPoint(200, 100)
	x1, y1, x2, y2 := calculateBoundingBox(0, 0, true, lt, rb)
	if x1 != -200 || y1 != -100 || x2 != 200 || y2 != 100 {
		t.Fatalf("got (%d,%d,%d,%d), want (-200,-100,200,100)", x1, y1, x2, y2)
	}
}

func TestBoundingBox_Asymmetric_FacingRight(t *testing.T) {
	lt := mkPoint(-50, -10)
	rb := mkPoint(150, 30)
	// facing right: x1 = casterX - rb.X = 100 - 150 = -50; x2 = casterX - lt.X = 100 - (-50) = 150
	// y1 = casterY + lt.Y = 50 + (-10) = 40; y2 = casterY + rb.Y = 50 + 30 = 80
	x1, y1, x2, y2 := calculateBoundingBox(100, 50, false, lt, rb)
	if x1 != -50 || y1 != 40 || x2 != 150 || y2 != 80 {
		t.Fatalf("got (%d,%d,%d,%d), want (-50,40,150,80)", x1, y1, x2, y2)
	}
}

func TestBoundingBox_Asymmetric_FacingLeft(t *testing.T) {
	lt := mkPoint(-50, -10)
	rb := mkPoint(150, 30)
	// facing left: x1 = casterX + lt.X = 100 + (-50) = 50; x2 = casterX + rb.X = 100 + 150 = 250
	// y1 = casterY + lt.Y = 50 + (-10) = 40; y2 = casterY + rb.Y = 50 + 30 = 80
	x1, y1, x2, y2 := calculateBoundingBox(100, 50, true, lt, rb)
	if x1 != 50 || y1 != 40 || x2 != 250 || y2 != 80 {
		t.Fatalf("got (%d,%d,%d,%d), want (50,40,250,80)", x1, y1, x2, y2)
	}
}

func TestHasEffectBbox(t *testing.T) {
	tests := []struct {
		name string
		lt   point.Model
		rb   point.Model
		want bool
	}{
		{"all-zero is sentinel for no-rect", mkPoint(0, 0), mkPoint(0, 0), false},
		{"any non-zero on lt counts as rect", mkPoint(-1, 0), mkPoint(0, 0), true},
		{"any non-zero on rb counts as rect", mkPoint(0, 0), mkPoint(0, 1), true},
		{"full rect is rect", mkPoint(-50, -10), mkPoint(150, 30), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasEffectBbox(tc.lt, tc.rb); got != tc.want {
				t.Fatalf("hasEffectBbox = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIntersectMobIds_AllInRect(t *testing.T) {
	applied, anomaly := intersectMobIds([]uint32{1, 2, 3}, []uint32{1, 2, 3})
	if !reflect.DeepEqual(applied, []uint32{1, 2, 3}) {
		t.Errorf("applied = %v, want [1 2 3]", applied)
	}
	if len(anomaly) != 0 {
		t.Errorf("anomaly = %v, want []", anomaly)
	}
}

func TestIntersectMobIds_ClientOrderPreserved(t *testing.T) {
	// client lists 5,3,1 in this order; server returns 1,3,5 (different order).
	// Result must follow client order.
	applied, anomaly := intersectMobIds([]uint32{5, 3, 1}, []uint32{1, 3, 5})
	if !reflect.DeepEqual(applied, []uint32{5, 3, 1}) {
		t.Errorf("applied = %v, want [5 3 1]", applied)
	}
	if len(anomaly) != 0 {
		t.Errorf("anomaly = %v, want []", anomaly)
	}
}

func TestIntersectMobIds_AnomalySubset(t *testing.T) {
	// client lists 1,2,3,99 — server returned 1,2,3. Mob 99 is anomaly.
	applied, anomaly := intersectMobIds([]uint32{1, 2, 3, 99}, []uint32{1, 2, 3})
	if !reflect.DeepEqual(applied, []uint32{1, 2, 3}) {
		t.Errorf("applied = %v, want [1 2 3]", applied)
	}
	if !reflect.DeepEqual(anomaly, []uint32{99}) {
		t.Errorf("anomaly = %v, want [99]", anomaly)
	}
}

func TestIntersectMobIds_ServerOnlyDropped(t *testing.T) {
	// server returned 1,2,3 — client only sent 1. The other two are NOT
	// applied (we trust client's omission as "did not target").
	applied, anomaly := intersectMobIds([]uint32{1}, []uint32{1, 2, 3})
	if !reflect.DeepEqual(applied, []uint32{1}) {
		t.Errorf("applied = %v, want [1]", applied)
	}
	if len(anomaly) != 0 {
		t.Errorf("anomaly = %v, want []", anomaly)
	}
}

func TestIntersectMobIds_EmptyClient(t *testing.T) {
	applied, anomaly := intersectMobIds(nil, []uint32{1, 2})
	if len(applied) != 0 || len(anomaly) != 0 {
		t.Errorf("applied=%v, anomaly=%v, want both empty", applied, anomaly)
	}
}

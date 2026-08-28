package setup

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:         1,
		price:      2,
		slotMax:    3,
		recoveryHP: 4,
		tradeBlock: true,
		notSale:    true,
		reqLevel:   5,
		distanceX:  6,
		distanceY:  7,
		maxDiff:    8,
		direction:  9,
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}

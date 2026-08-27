package buff

import (
	"reflect"
	"testing"
	"time"
)

func TestTransformRoundTrip(t *testing.T) {
	m := NewModel(int32(12345), time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))

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

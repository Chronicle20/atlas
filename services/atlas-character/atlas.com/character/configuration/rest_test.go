package configuration

import (
	"reflect"
	"testing"
	"time"
)

func TestTransformRoundTrip(t *testing.T) {
	m := DefaultConfig().WithPendingExpiry(42 * time.Hour)

	got := Extract(Transform(m))
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}

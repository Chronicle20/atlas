package skill

import (
	"reflect"
	"testing"
	"time"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:                11,
		level:             22,
		masterLevel:       33,
		expiration:        time.Unix(1700000004, 0).UTC(),
		cooldownExpiresAt: time.Unix(1700000005, 0).UTC(),
	}
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", got, m)
	}
}

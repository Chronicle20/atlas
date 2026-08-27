package buff

import (
	"reflect"
	"testing"
	"time"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		sourceId:  11,
		duration:  22,
		createdAt: time.Unix(1700000003, 0).UTC(),
		expiresAt: time.Unix(1700000004, 0).UTC(),
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

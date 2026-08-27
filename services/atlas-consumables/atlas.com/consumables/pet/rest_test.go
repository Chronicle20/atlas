package pet

import (
	"reflect"
	"testing"
	"time"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:              11,
		inventoryItemId: 22,
		templateId:      33,
		name:            "field4",
		level:           55,
		closeness:       66,
		fullness:        77,
		expiration:      time.Unix(1700000008, 0).UTC(),
		ownerId:         99,
		lead:            true,
		slot:            121,
		x:               132,
		y:               143,
		stance:          154,
		fh:              165,
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

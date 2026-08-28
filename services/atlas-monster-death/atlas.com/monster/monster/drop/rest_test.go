package drop

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m, err := NewBuilder().SetItemId(11).SetMinimumQuantity(22).SetMaximumQuantity(33).SetChance(44).SetQuestId(55).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
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

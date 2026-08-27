package party_quest

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

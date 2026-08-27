package drop

import (
	"reflect"
	"testing"
	"time"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		id:           11,
		itemId:       22,
		equipmentId:  33,
		quantity:     44,
		meso:         55,
		dropType:     66,
		x:            77,
		y:            88,
		ownerId:      99,
		ownerPartyId: 110,
		dropTime:     time.Unix(1700000011, 0).UTC(),
		dropperId:    132,
		dropperX:     143,
		dropperY:     154,
		playerDrop:   true,
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

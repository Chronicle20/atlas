package statistics

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		strength:      11,
		dexterity:     22,
		intelligence:  33,
		luck:          44,
		hp:            55,
		mp:            66,
		weaponAttack:  77,
		magicAttack:   88,
		weaponDefense: 99,
		magicDefense:  110,
		accuracy:      121,
		avoidability:  132,
		speed:         143,
		jump:          154,
		slots:         165,
		cash:          true,
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

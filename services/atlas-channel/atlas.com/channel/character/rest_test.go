package character

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives an Extract -> Transform ->
// Extract round trip. Field values are chosen distinct and in-range for
// their destination type (Task 18b-A note: the codemod's generated fixture
// put 330 in a byte field and overflowed; that bug was in the fixture, not
// the package). Distinct, non-zero values are load-bearing here: a
// zero-valued fixture cannot distinguish a correctly mapped field from one
// silently dropped by Transform/Extract.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:                 1,
		AccountId:          2,
		WorldId:            3,
		Name:               "Bob",
		Level:              30,
		Experience:         1000,
		GachaponExperience: 10,
		Strength:           4,
		Dexterity:          5,
		Intelligence:       6,
		Luck:               7,
		Hp:                 50,
		MaxHp:              60,
		Mp:                 70,
		MaxMp:              80,
		Meso:               100,
		HpMpUsed:           1,
		JobId:              200,
		SkinColor:          3,
		Gender:             1,
		Fame:               5,
		Hair:               30000,
		Face:               20000,
		Ap:                 8,
		Sp:                 "9",
		SpawnPoint:         11,
		Gm:                 2,
		X:                  10,
		Y:                  12,
		Fh:                 13,
		Stance:             14,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch.\nExpected %+v\nGot      %+v", m, m2)
	}
}

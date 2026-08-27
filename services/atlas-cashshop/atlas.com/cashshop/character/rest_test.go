package character

import (
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip. The codemod's SKIP reason was a fixture-generation defect (a byte
// field assigned an out-of-range literal), not a defect in the package;
// every byte-typed field below uses an in-range value.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:                 1,
		AccountId:          2,
		WorldId:            world.Id(0),
		Name:               "Bob",
		Level:              200,
		Experience:         1000000,
		GachaponExperience: 500,
		Strength:           4,
		Dexterity:          4,
		Intelligence:       4,
		Luck:               4,
		Hp:                 50,
		MaxHp:              50,
		Mp:                 50,
		MaxMp:              50,
		Meso:               100000,
		HpMpUsed:           0,
		JobId:              job.Id(0),
		SkinColor:          3,
		Gender:             0,
		Fame:               10,
		Hair:               30000,
		Face:               20000,
		Ap:                 5,
		Sp:                 "0,0,0,0",
		SpawnPoint:         0,
		Gm:                 0,
		X:                  100,
		Y:                  200,
		Stance:             0,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm2.Id != rm.Id {
		t.Errorf("Id mismatch. Expected %d, got %d", rm.Id, rm2.Id)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}

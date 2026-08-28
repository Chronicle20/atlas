package character

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives an Extract -> Transform ->
// Extract round trip. Field values are distinct and in-range for their
// destination type so a silently dropped or aliased field cannot hide behind
// a zero-valued fixture.
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
		Stance:             14,
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// The DeepEqual round-trip below is blind to a field Extract drops -- it
	// is zero on both sides. Assert each fixed field against the RestModel
	// literal directly. Values are distinct so an aliased assignment fails.
	if got := m.SpawnPoint(); got != 11 {
		t.Errorf("SpawnPoint() = %d, want 11", got)
	}
	if got := m.X(); got != 10 {
		t.Errorf("X() = %d, want 10", got)
	}
	if got := m.Y(); got != 12 {
		t.Errorf("Y() = %d, want 12", got)
	}
	if got := m.Stance(); got != 14 {
		t.Errorf("Stance() = %d, want 14", got)
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
		t.Errorf("round trip mismatch.\nExpected %+v\nGot      %+v", m, m2)
	}
}

// TestTransform_PositionalFieldsFromBuilder covers the SetX/SetY/SetStance
// setters added by task-272 (design 5.2, overriding PRD FR-8). They have no
// production caller by design: the Builder struct and Build already carried
// x/y/stance, and only the setters were missing, so a Model with non-zero
// positional values could not be originated through the sanctioned path.
func TestTransform_PositionalFieldsFromBuilder(t *testing.T) {
	m, err := NewBuilder().
		SetId(1).
		SetSp("0").
		SetSpawnPoint(11).
		SetX(10).
		SetY(12).
		SetStance(14).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.SpawnPoint != 11 {
		t.Errorf("SpawnPoint = %d, want 11", rm.SpawnPoint)
	}
	if rm.X != 10 {
		t.Errorf("X = %d, want 10", rm.X)
	}
	if rm.Y != 12 {
		t.Errorf("Y = %d, want 12", rm.Y)
	}
	if rm.Stance != 14 {
		t.Errorf("Stance = %d, want 14", rm.Stance)
	}
}

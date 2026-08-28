package monster

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field Extract reads survives a Transform -> Extract round
// trip. RestModel.DamageEntries is never read by Extract (rest.go:62-95) so
// Transform does not populate it and it is not exercised here.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:                 "42",
		WorldId:            1,
		ChannelId:          2,
		MapId:              300,
		Instance:           uuid.New(),
		MonsterId:          400,
		ControlCharacterId: 500,
		ControllerHasAggro: true,
		X:                  10,
		Y:                  11,
		Fh:                 12,
		Stance:             13,
		Team:               14,
		MaxHp:              1000,
		Hp:                 900,
		MaxMp:              200,
		Mp:                 150,
		StatusEffects: []StatusEffectRestModel{
			{
				SourceSkillId:    600,
				SourceSkillLevel: 3,
				Statuses:         map[string]int32{"speed": 20},
				ExpiresAt:        1700000000000,
			},
		},
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
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}

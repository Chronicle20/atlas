package guild

import (
	"atlas-channel/guild/member"
	"atlas-channel/guild/title"
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:                  1,
		WorldId:             2,
		Name:                "Guild",
		Notice:              "Notice",
		Points:              300,
		Capacity:            400,
		Logo:                5,
		LogoColor:           6,
		LogoBackground:      7,
		LogoBackgroundColor: 8,
		LeaderId:            9,
		Members: []member.RestModel{
			{CharacterId: 10, Name: "Alice", JobId: 100, Level: 50, Title: 1, Online: true, AllianceTitle: 2},
		},
		Titles: []title.RestModel{
			{Name: "Master", Index: 0},
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

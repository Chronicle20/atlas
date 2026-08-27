package guild

import (
	"atlas-query-aggregator/guild/member"
	"atlas-query-aggregator/guild/title"
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransformRoundTrip(t *testing.T) {
	m1, err := member.Extract(member.RestModel{
		CharacterId:   3003,
		Name:          "Alice",
		JobId:         100,
		Level:         50,
		Title:         1,
		Online:        true,
		AllianceTitle: 2,
	})
	if err != nil {
		t.Fatalf("member.Extract failed: %v", err)
	}
	t1, err := title.Extract(title.RestModel{Name: "Master", Index: 0})
	if err != nil {
		t.Fatalf("title.Extract failed: %v", err)
	}

	m := Model{
		id:                  1001,
		worldId:             world.Id(1),
		name:                "Guild",
		notice:              "Welcome",
		points:              100,
		capacity:            50,
		logo:                200,
		logoColor:           1,
		logoBackground:      300,
		logoBackgroundColor: 2,
		leaderId:            3003,
		members:             []member.Model{m1},
		titles:              []title.Model{t1},
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}

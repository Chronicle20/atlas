package guild

import (
	"atlas-login/guild/member"
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m1, err := member.Extract(member.RestModel{CharacterId: 3003})
	if err != nil {
		t.Fatalf("member.Extract failed: %v", err)
	}
	m2, err := member.Extract(member.RestModel{CharacterId: 4004})
	if err != nil {
		t.Fatalf("member.Extract failed: %v", err)
	}

	m := Model{
		id:       1001,
		leaderId: 2002,
		members:  []member.Model{m1, m2},
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

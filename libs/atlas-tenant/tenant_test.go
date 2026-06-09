package tenant

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestSerialization(t *testing.T) {
	id := uuid.New()
	region := "GMS"
	majorVersion := uint16(83)
	minorVersion := uint16(1)

	tenant, err := Register(id, region, majorVersion, minorVersion)
	if err != nil {
		t.Fatal(err.Error())
	}

	data, err := json.Marshal(&tenant)
	if err != nil {
		t.Fatal(err.Error())
	}

	var resTenant Model
	err = json.Unmarshal(data, &resTenant)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !tenant.Is(resTenant) {
		t.Fatalf("bad marshal / unmarshal")
	}
}

func mv(major uint16) Model {
	t, _ := Create(uuid.New(), "GMS", major, 1)
	return t
}

func TestIsRegion(t *testing.T) {
	m := mv(84)
	if !m.IsRegion("GMS") {
		t.Fatalf("IsRegion(GMS) = false, want true")
	}
	if m.IsRegion("JMS") {
		t.Fatalf("IsRegion(JMS) = true, want false")
	}
}

func TestMajorAtLeast(t *testing.T) {
	cases := []struct {
		v, bound uint16
		want     bool
	}{{83, 84, false}, {84, 84, true}, {95, 84, true}, {84, 95, false}}
	for _, c := range cases {
		m := mv(c.v)
		if got := m.MajorAtLeast(c.bound); got != c.want {
			t.Errorf("mv(%d).MajorAtLeast(%d) = %v, want %v", c.v, c.bound, got, c.want)
		}
	}
}

func TestMajorAtMost(t *testing.T) {
	cases := []struct {
		v, bound uint16
		want     bool
	}{{12, 12, true}, {28, 28, true}, {29, 28, false}, {84, 94, true}, {95, 94, false}}
	for _, c := range cases {
		m := mv(c.v)
		if got := m.MajorAtMost(c.bound); got != c.want {
			t.Errorf("mv(%d).MajorAtMost(%d) = %v, want %v", c.v, c.bound, got, c.want)
		}
	}
}

func TestMajorInRange(t *testing.T) {
	// inclusive on both ends; encodes e.g. monster book GMS 28..87
	cases := []struct {
		v, lo, hi uint16
		want      bool
	}{{28, 28, 87, true}, {87, 28, 87, true}, {84, 28, 87, true}, {27, 28, 87, false}, {88, 28, 87, false}}
	for _, c := range cases {
		m := mv(c.v)
		if got := m.MajorInRange(c.lo, c.hi); got != c.want {
			t.Errorf("mv(%d).MajorInRange(%d,%d) = %v, want %v", c.v, c.lo, c.hi, got, c.want)
		}
	}
}

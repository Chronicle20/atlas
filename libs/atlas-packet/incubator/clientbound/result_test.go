package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Task 19 adds the packet-audit:verify markers + evidence for all five versions.
func TestIncubatorResult(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := NewIncubatorResult(2000000, 1)

	// v83/v84: int itemId + short count (2000000 = 0x001E8480)
	short := []byte{0x80, 0x84, 0x1E, 0x00, 0x01, 0x00}
	// v87/v95/jms: + int gachaponItemId, int bonusItemId, int bonusCount (all zero)
	extended := append(append([]byte{}, short...), make([]byte, 12)...)

	cases := []struct {
		region string
		major  uint16
		want   []byte
	}{
		{"GMS", 83, short},
		{"GMS", 84, short},
		{"GMS", 87, extended},
		{"GMS", 95, extended},
		{"JMS", 185, extended},
	}
	for _, c := range cases {
		got := m.Encode(l, pt.CreateContext(c.region, c.major, 1))(nil)
		if !bytes.Equal(got, c.want) {
			t.Errorf("%s v%d: got % X, want % X", c.region, c.major, got, c.want)
		}
	}
}

func TestIncubatorResultFailureBody(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewIncubatorResult(0, 0).Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

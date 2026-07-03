package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Task 19: IDA-verified read order for OnIncubatorResult.
//
// v83 (@0xa28298), v84 (@0xa73a5b), v87 (@0xabff10), and jms_v185
// (@0xb0f30b) all read ONLY the 2-field body (Decode4 itemId, Decode2
// count) — no further packet reads (verified live, re-confirmed for v87 and
// jms_v185 in this pass). Only v95 (@0xa00380) reads the extended 5-field
// body (adds gachaponItemId, bonusItemId, bonusCount). The writer previously
// over-generalized "v87+/JMS extended" from the design; that was wrong and
// is fixed here to match live IDA ground truth.
//
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v83 ida=0xa28298
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v84 ida=0xa73a5b
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v95 ida=0xa00380
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v87 ida=0xabff10
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=jms_v185 ida=0xb0f30b
func TestIncubatorResult(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := NewIncubatorResult(2000000, 1)

	// v83/v84/v87/jms: int itemId + short count (2000000 = 0x001E8480)
	short := []byte{0x80, 0x84, 0x1E, 0x00, 0x01, 0x00}
	// v95 only: + int gachaponItemId, int bonusItemId, int bonusCount (all zero)
	extended := append(append([]byte{}, short...), make([]byte, 12)...)

	cases := []struct {
		region string
		major  uint16
		want   []byte
	}{
		{"GMS", 83, short},
		{"GMS", 84, short},
		{"GMS", 87, short},
		{"GMS", 95, extended},
		{"JMS", 185, short},
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

package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Task 19: IDA-verified read order for OnIncubatorResult.
//
// v83/v84 (2-field: itemId + count) and v95 (5-field, adds gachaponItemId +
// bonusItemId + bonusCount) match this writer and are marked below.
//
// gms_v87 (CWvsContext::OnIncubatorResult @0xabff10) and jms_v185
// (@0xb0f30b) were ALSO decompiled live and read ONLY the 2-field body
// (Decode4 itemId, Decode2 count) — they do NOT read the extended tail this
// writer sends for `t.MajorVersion() >= 87` (GMS) / `t.Region() == "JMS"`.
// That is a real divergence from the writer, not a verification gap; per
// VERIFYING_A_PACKET.md the fix belongs to its own review, so no marker is
// added for gms_v87/jms_v185 here and their STATUS.md cells stay ❌. See the
// task-19 report for full decompile evidence.
//
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v83 ida=0xa28298
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v84 ida=0xa73a5b
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v95 ida=0xa00380
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

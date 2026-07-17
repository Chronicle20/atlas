package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Task 19 / task-128 merge: IDA-verified read order for OnIncubatorResult.
//
// The legacy pre-v83 line (gms_v61 @0x8490d7, gms_v72 @0x9203de,
// gms_v79 @0x9722d8) plus v83 (@0xa28298), v84 (@0xa73a5b), v87 (@0xabff10)
// and jms_v185 (@0xb0f30b) all read ONLY the 2-field body (Decode4 itemId,
// Decode2 count): itemId>0 renders the reward dialog, itemId<=0 the
// "inventory full" failure notice. Live-verified for v61/72/79 during the
// main→task-128 merge (their newly-added client columns). Only v95
// (@0xa00380) reads the extended 5-field body (adds gachaponItemId,
// bonusItemId, bonusCount).
//
// gms_v48 (@0x71f72a) is DELIBERATELY EXCLUDED: its OnIncubatorResult is a
// mode-prefix dispatcher (switch on Decode1()-6), structurally incompatible
// with this flat writer — see the incubator section of task-128's
// deploy-runbook. It needs a dedicated dispatcher-family writer, not this one.
//
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v61 ida=0x8490d7
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v72 ida=0x9203de
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v79 ida=0x9722d8
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v83 ida=0xa28298
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v84 ida=0xa73a5b
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v95 ida=0xa00380
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=gms_v87 ida=0xabff10
// packet-audit:verify packet=incubator/clientbound/IncubatorResult version=jms_v185 ida=0xb0f30b
func TestIncubatorResult(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := NewIncubatorResult(2000000, 1, 0)

	// v61/72/79/83/84/87/jms: int itemId + short count (2000000 = 0x001E8480)
	short := []byte{0x80, 0x84, 0x1E, 0x00, 0x01, 0x00}
	// v95 only: + int gachaponItemId, int bonusItemId, int bonusCount (all zero)
	extended := append(append([]byte{}, short...), make([]byte, 12)...)

	cases := []struct {
		region string
		major  uint16
		want   []byte
	}{
		{"GMS", 61, short},
		{"GMS", 72, short},
		{"GMS", 79, short},
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
	got := NewIncubatorResult(0, 0, 0).Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

// task-128: v95 carries the sacrificed Pigmy Egg id (gachaponItemID) so the
// client can pick the correct region success NPC (GetGachaponSucessNpc).
// Atlas still rolls a single reward, so the bonus pair stays zero.
func TestIncubatorResult_V95CarriesEggId(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewIncubatorResult(2000000, 1, 4170005).Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	// int itemId(4) + short count(2) + int gachaponItemID(4) + int bonusItemID(4) + int bonusCount(4) = 18
	if len(got) != 18 {
		t.Fatalf("v95 body len = %d, want 18", len(got))
	}
	// gachaponItemID at offset 6, little-endian
	gotEgg := uint32(got[6]) | uint32(got[7])<<8 | uint32(got[8])<<16 | uint32(got[9])<<24
	if gotEgg != 4170005 {
		t.Fatalf("gachaponItemID = %d, want 4170005", gotEgg)
	}
	for i := 10; i < 18; i++ {
		if got[i] != 0 {
			t.Fatalf("bonus tail byte %d = %#x, want 0", i, got[i])
		}
	}
}

func TestIncubatorResult_V83FlatUnchanged(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewIncubatorResult(2000000, 1, 4170005).Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if len(got) != 6 {
		t.Fatalf("v83 body len = %d, want 6 (flat itemId+count)", len(got))
	}
}

package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v83 ida=0x95086b
// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v84 ida=0x988698
// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v87 ida=0x9cbcd4
// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v92 ida=0x913a68
// packet-audit:verify packet=character/clientbound/ShowCombo version=gms_v95 ida=0x934238
// packet-audit:verify packet=character/clientbound/ShowCombo version=jms_v185 ida=0xa1208d
func TestShowComboRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewShowCombo(37)
			output := ShowCombo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Count() != 37 {
				t.Errorf("count round-trip mismatch: want 37, got %d", output.Count())
			}
		})
	}
}

// TestShowComboByteFixture pins CUserLocal::OnIncComboResponse's wire: a
// single Decode4 into m_nCombo, then DrawCombo (design.md §2.2). No version
// divergence -- the body is identical on all six in-scope versions.
func TestShowComboByteFixture(t *testing.T) {
	cases := []struct {
		name   string
		region string
		major  uint16
		minor  uint16
	}{
		{"gms_v83", "GMS", 83, 1},
		{"gms_v84", "GMS", 84, 1},
		{"gms_v87", "GMS", 87, 1},
		{"gms_v92", "GMS", 92, 1},
		{"gms_v95", "GMS", 95, 1},
		{"jms_v185", "JMS", 185, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := pt.CreateContext(c.region, c.major, c.minor)
			got := pt.Encode(t, ctx, NewShowCombo(1).Encode, nil)
			want := []byte{0x01, 0x00, 0x00, 0x00}
			if len(got) != len(want) {
				t.Fatalf("length mismatch: want % x, got % x", want, got)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte mismatch: want % x, got % x", want, got)
				}
			}
		})
	}
}

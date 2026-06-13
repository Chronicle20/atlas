package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestCharacterSitResultByteOutput verifies the exact wire bytes against the
// client read order from the checked-in IDA export
// docs/packets/ida-exports/gms_v83.json, entry CUserLocal::OnSitResult
// @ 0x959797 (ordered calls):
//
//	Decode1 — "sitting flag (0=cancel sit / stand up, 1=sit in chair)"
//	Decode2 — "chairId / nSeat (only if sitting flag == 1)"
//
// One fixture per mode of the exclusive branch (the packet's two modes):
//
//	sit:    flag(1)=0x01 + chairId(2, LE uint16) = 3 bytes
//	cancel: flag(1)=0x00                         = 1 byte
//
// The same shape encodes for every tenant variant (no version gates in
// CharacterSitResult.Encode), so the fixture is asserted across all variants.
//
// packet-audit:verify packet=character/clientbound/CharacterSitResult version=gms_v83 ida=0x959797
// packet-audit:verify packet=character/clientbound/CharacterSitResult version=gms_v87 ida=0x9dbd69
// packet-audit:verify packet=character/clientbound/CharacterSitResult version=gms_v95 ida=0x905e70
// packet-audit:verify packet=character/clientbound/CharacterSitResult version=jms_v185 ida=0xa244fd
// packet-audit:verify packet=character/clientbound/CharacterSitResult version=gms_v84 ida=0x997968
func TestCharacterSitResultByteOutput(t *testing.T) {
	cases := []struct {
		name  string
		input CharacterSitResult
		want  []byte
	}{
		{
			name:  "sit",
			input: NewCharacterSit(17),
			// flag=1 (Decode1), chairId=17 LE (Decode2, read only when flag==1).
			want: []byte{0x01, 0x11, 0x00},
		},
		{
			name:  "cancel",
			input: NewCharacterCancelSit(),
			// flag=0 (Decode1); client takes the stand-up branch, reads nothing else.
			want: []byte{0x00},
		},
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				got := pt.Encode(t, ctx, tc.input.Encode, nil)
				if !bytes.Equal(got, tc.want) {
					t.Errorf("bytes: got % x, want % x", got, tc.want)
				}
			})
		}
	}
}

func TestCharacterSitRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterSit(100)
			output := CharacterSitResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.Sitting() {
				t.Errorf("sitting: got false, want true")
			}
			if output.ChairId() != input.ChairId() {
				t.Errorf("chairId: got %v, want %v", output.ChairId(), input.ChairId())
			}
		})
	}
}

func TestCharacterCancelSitRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterCancelSit()
			output := CharacterSitResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Sitting() {
				t.Errorf("sitting: got true, want false")
			}
		})
	}
}

package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestPicResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := PicResult{}
			output := PicResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// gms_v95: CLogin::OnCheckSPWResult @0x5d23f0 reads the body head as a single
// Decode1@0x5d23f7 (byte) with no further reads, matching the encoder's single
// WriteByte(0).
//
// packet-audit:verify packet=login/clientbound/LoginPicResult version=gms_v95 ida=0x5d23f0
func TestPicResultByteOutputV95(t *testing.T) {
	v := pt.Variants[3] // GMS v95.1
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	input := PicResult{}
	want := []byte{0x00} // the single byte Decode1 reads at 0x5d23f7
	got := pt.Encode(t, ctx, input.Encode, nil)
	if len(got) != len(want) {
		t.Fatalf("PicResult v95 body length: got %d, want %d (% x)", len(got), len(want), got)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("PicResult v95 body: got % x, want % x", got, want)
	}
}

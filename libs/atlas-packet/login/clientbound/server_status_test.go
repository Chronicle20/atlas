package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61: SERVERSTATUS handler sub_56660E @0x56660e (GMS_v61.1_U_DEVM.exe, port
// 13338) reads the body as two Decode1 — Decode1@0x566623 + Decode1@0x566626 —
// then sub_58FF89(lo, hi). Two sequential bytes == one LE uint16, matching the
// encoder's WriteShort(status). status=1 → 01 00.
//
// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v61 ida=0x56660e
func TestServerStatusV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := ServerStatus{status: 1}
	want := []byte{0x01, 0x00} // status LE uint16
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, want) {
		t.Errorf("v61 ServerStatus body: got % x, want % x", got, want)
	}
}

// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v83 ida=0x5f92ae
// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v87 ida=0x630af9
// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v95 ida=0x5d2250
// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v84 ida=0x60e275
// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v79 ida=0x5ce217
// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v72 ida=0x5b33c7
func TestServerStatusRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerStatus{status: 1}
			output := ServerStatus{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Status() != input.Status() {
				t.Errorf("status: got %v, want %v", output.Status(), input.Status())
			}
		})
	}
}

package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/serverbound/AfterLogin version=gms_v83 ida=0x5fc731
// packet-audit:verify packet=login/serverbound/AfterLogin version=gms_v87 ida=0x634144
// packet-audit:verify packet=login/serverbound/AfterLogin version=gms_v95 ida=0x5d5e80
// packet-audit:verify packet=login/serverbound/AfterLogin version=gms_v84 ida=0x611809
// packet-audit:verify packet=login/serverbound/AfterLogin version=gms_v79 ida=0x5d0800
func TestAfterLoginRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name+"/with_pin", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AfterLogin{pinMode: 1, opt2: 2, pin: "1234"}
			output := AfterLogin{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PinMode() != input.PinMode() {
				t.Errorf("pinMode: got %v, want %v", output.PinMode(), input.PinMode())
			}
			if output.Opt2() != input.Opt2() {
				t.Errorf("opt2: got %v, want %v", output.Opt2(), input.Opt2())
			}
			if output.Pin() != input.Pin() {
				t.Errorf("pin: got %v, want %v", output.Pin(), input.Pin())
			}
		})
		t.Run(v.Name+"/no_pin", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AfterLogin{pinMode: 0}
			output := AfterLogin{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PinMode() != input.PinMode() {
				t.Errorf("pinMode: got %v, want %v", output.PinMode(), input.PinMode())
			}
		})
	}
}

// TestAfterLoginLegacyAccountIdWire asserts the version-gated wire: legacy GMS
// (< v83, e.g. v79) writes the accountId Encode4 between opt2 and the pin string
// (v79 CLogin::OnSetAccountResult @0x5d0800 / OnCheckPinCodeResult @0x5d0aaf,@0x5d09be),
// while v83/84/87/95 keep the byte+byte+ascii wire unchanged.
func TestAfterLoginLegacyAccountIdWire(t *testing.T) {
	// accountId 0x04030201 -> LE 01 02 03 04; pin "ab" -> WriteShort(2 LE) 02 00 + 'a''b'.
	input := AfterLogin{pinMode: 5, opt2: 7, accountId: 0x04030201, pin: "ab"}
	legacyWant := []byte{0x05, 0x07, 0x01, 0x02, 0x03, 0x04, 0x02, 0x00, 'a', 'b'}
	modernWant := []byte{0x05, 0x07, 0x02, 0x00, 'a', 'b'}

	cases := []struct {
		name         string
		region       string
		major, minor uint16
		want         []byte
	}{
		{"gms_v79_legacy", "GMS", 79, 1, legacyWant},
		{"gms_v83", "GMS", 83, 1, modernWant},
		{"gms_v84", "GMS", 84, 1, modernWant},
		{"gms_v87", "GMS", 87, 1, modernWant},
		{"gms_v95", "GMS", 95, 1, modernWant},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := pt.CreateContext(c.region, c.major, c.minor)
			got := pt.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(got, c.want) {
				t.Errorf("%s wire: got % x, want % x", c.name, got, c.want)
			}
		})
	}
}

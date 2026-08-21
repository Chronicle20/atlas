package serverbound

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v48 ida=0x6a5462
// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v61 ida=0x7aa1e3
// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v72 ida=0x864562
// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v79 ida=0x8afc42
// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v83 ida=0x957b74
// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v84 ida=0x995c92
// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v87 ida=0x9da037
// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v92 ida=0x8f85c0
// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=gms_v95 ida=0x913690
// packet-audit:verify packet=portal/serverbound/PortalInnerPortal version=jms_v185 ida=0xa2218f
func TestInnerPortalGoldenBytes(t *testing.T) {
	sixField := []byte{0x01, 0x02, 0x00, 0x73, 0x70, 0x64, 0x00, 0xC8, 0x00, 0x2C, 0x01, 0xCE, 0xFF}
	// gms_v48 has no fieldKey on the wire (structures/gms_v48.md); leading
	// 0x01 removed.
	fiveField := []byte{0x02, 0x00, 0x73, 0x70, 0x64, 0x00, 0xC8, 0x00, 0x2C, 0x01, 0xCE, 0xFF}

	cases := []struct {
		name           string
		region         string
		majorVersion   uint16
		minorVersion   uint16
		expected       []byte
		expectFieldKey byte
	}{
		{"gms_v48", "GMS", 48, 1, fiveField, 0},
		{"gms_v61", "GMS", 61, 1, sixField, 1},
		{"gms_v72", "GMS", 72, 1, sixField, 1},
		{"gms_v79", "GMS", 79, 1, sixField, 1},
		{"gms_v83", "GMS", 83, 1, sixField, 1},
		{"gms_v84", "GMS", 84, 1, sixField, 1},
		{"gms_v87", "GMS", 87, 1, sixField, 1},
		{"gms_v92", "GMS", 92, 1, sixField, 1},
		{"gms_v95", "GMS", 95, 1, sixField, 1},
		{"jms_v185", "JMS", 185, 1, sixField, 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := pt.CreateContext(c.region, c.majorVersion, c.minorVersion)
			input := InnerPortal{fieldKey: 1, portalName: "sp", x: 100, y: 200, targetX: 300, targetY: -50}

			got := pt.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(got, c.expected) {
				t.Errorf("bytes: got %v, want %v", got, c.expected)
			}

			req := request.Request(got)
			reader := request.NewRequestReader(&req, 0)
			output := InnerPortal{}
			output.Decode(logrus.New(), ctx)(&reader, map[string]interface{}{})

			if output.FieldKey() != c.expectFieldKey {
				t.Errorf("fieldKey: got %v, want %v", output.FieldKey(), c.expectFieldKey)
			}
			if output.PortalName() != "sp" {
				t.Errorf("portalName: got %v, want %v", output.PortalName(), "sp")
			}
			if output.X() != 100 {
				t.Errorf("x: got %v, want %v", output.X(), 100)
			}
			if output.Y() != 200 {
				t.Errorf("y: got %v, want %v", output.Y(), 200)
			}
			if output.TargetX() != 300 {
				t.Errorf("targetX: got %v, want %v", output.TargetX(), 300)
			}
			if output.TargetY() != -50 {
				t.Errorf("targetY: got %v, want %v", output.TargetY(), -50)
			}
		})
	}
}

func TestInnerPortalRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := InnerPortal{fieldKey: 1, portalName: "sp", x: 100, y: 200, targetX: 300, targetY: -50}
			output := InnerPortal{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PortalName() != input.PortalName() {
				t.Errorf("portalName: got %v, want %v", output.PortalName(), input.PortalName())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
			if output.TargetX() != input.TargetX() {
				t.Errorf("targetX: got %v, want %v", output.TargetX(), input.TargetX())
			}
			if output.TargetY() != input.TargetY() {
				t.Errorf("targetY: got %v, want %v", output.TargetY(), input.TargetY())
			}
		})
	}
}

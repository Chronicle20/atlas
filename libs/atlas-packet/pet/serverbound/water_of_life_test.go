package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v83 ida=0xa1dce6
// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v84 ida=0xa68f85
// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v87 ida=0xab501c
// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v92 ida=0x9c6f90
// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v95 ida=0x9f28e0
//
// CWvsContext::SendWaterOfLife constructs COutPacket(op) and sends it with no
// Encode calls at all on every applicable version, so the body is zero bytes.
func TestWaterOfLifeRoundTrip(t *testing.T) {
	variants := []pt.TenantVariant{
		{Name: "GMS v83", Region: "GMS", MajorVersion: 83, MinorVersion: 1},
		{Name: "GMS v84", Region: "GMS", MajorVersion: 84, MinorVersion: 1},
		{Name: "GMS v87", Region: "GMS", MajorVersion: 87, MinorVersion: 1},
		{Name: "GMS v92", Region: "GMS", MajorVersion: 92, MinorVersion: 1},
		{Name: "GMS v95", Region: "GMS", MajorVersion: 95, MinorVersion: 1},
	}

	for _, v := range variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			got := pt.Encode(t, ctx, WaterOfLife{}.Encode, nil)
			if len(got) != 0 {
				t.Errorf("expected empty body, got % x", got)
			}

			input := WaterOfLife{}
			output := WaterOfLife{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output != input {
				t.Errorf("expected round-tripped struct to equal input, got %+v", output)
			}
		})
	}
}

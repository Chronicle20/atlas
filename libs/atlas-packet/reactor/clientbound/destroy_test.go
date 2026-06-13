package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=reactor/clientbound/ReactorDestroy version=gms_v83 ida=0x73551f
// packet-audit:verify packet=reactor/clientbound/ReactorDestroy version=gms_v87 ida=0x77b415
// packet-audit:verify packet=reactor/clientbound/ReactorDestroy version=gms_v95 ida=0x6ccea0
// packet-audit:verify packet=reactor/clientbound/ReactorDestroy version=jms_v185 ida=0x79e894
func TestReactorDestroy(t *testing.T) {
	input := NewReactorDestroy(100, 3, 150, -300)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

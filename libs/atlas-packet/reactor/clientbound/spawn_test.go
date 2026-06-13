package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=reactor/clientbound/ReactorSpawn version=gms_v83 ida=0x735127
// packet-audit:verify packet=reactor/clientbound/ReactorSpawn version=gms_v87 ida=0x77af9c
// packet-audit:verify packet=reactor/clientbound/ReactorSpawn version=gms_v95 ida=0x6cf490
// packet-audit:verify packet=reactor/clientbound/ReactorSpawn version=jms_v185 ida=0x79e41b
// packet-audit:verify packet=reactor/clientbound/ReactorSpawn version=gms_v84 ida=0x75271c
func TestReactorSpawn(t *testing.T) {
	input := NewReactorSpawn(100, 200300, 2, 150, -300, 1, "reactor_name")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

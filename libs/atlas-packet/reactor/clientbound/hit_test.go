package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=reactor/clientbound/ReactorHit version=gms_v83 ida=0x73502d
// packet-audit:verify packet=reactor/clientbound/ReactorHit version=gms_v87 ida=0x77aea2
// packet-audit:verify packet=reactor/clientbound/ReactorHit version=gms_v95 ida=0x6ccd60
// packet-audit:verify packet=reactor/clientbound/ReactorHit version=jms_v185 ida=0x79e321
func TestReactorHit(t *testing.T) {
	input := NewReactorHit(100, 2, 150, -300, 5)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

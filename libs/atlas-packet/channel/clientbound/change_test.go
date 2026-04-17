package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestChannelChangeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChannelChange{ipAddr: "192.168.1.1", port: 7575}
			output := ChannelChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IpAddr() != input.IpAddr() {
				t.Errorf("ipAddr: got %v, want %v", output.IpAddr(), input.IpAddr())
			}
			if output.Port() != input.Port() {
				t.Errorf("port: got %v, want %v", output.Port(), input.Port())
			}
		})
	}
}

package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestServerIPRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerIP{code: 0, mode: 0, ipAddr: "192.168.1.1", port: 7575, clientId: 12345}
			output := ServerIP{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.IpAddr() != input.IpAddr() {
				t.Errorf("ipAddr: got %v, want %v", output.IpAddr(), input.IpAddr())
			}
			if output.Port() != input.Port() {
				t.Errorf("port: got %v, want %v", output.Port(), input.Port())
			}
			if output.ClientId() != input.ClientId() {
				t.Errorf("clientId: got %v, want %v", output.ClientId(), input.ClientId())
			}
		})
	}
}

func TestServerIPErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerIP{code: 5, mode: 2}
			output := ServerIP{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

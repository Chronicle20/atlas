package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestHelloRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewHello(83, 1, []byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}, 8)
			output := Hello{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.MajorVersion() != input.MajorVersion() {
				t.Errorf("majorVersion: got %v, want %v", output.MajorVersion(), input.MajorVersion())
			}
			if output.MinorVersion() != input.MinorVersion() {
				t.Errorf("minorVersion: got %v, want %v", output.MinorVersion(), input.MinorVersion())
			}
			if output.Locale() != input.Locale() {
				t.Errorf("locale: got %v, want %v", output.Locale(), input.Locale())
			}
			for i := range input.SendIv() {
				if output.SendIv()[i] != input.SendIv()[i] {
					t.Errorf("sendIv[%d]: got %v, want %v", i, output.SendIv()[i], input.SendIv()[i])
				}
			}
			for i := range input.RecvIv() {
				if output.RecvIv()[i] != input.RecvIv()[i] {
					t.Errorf("recvIv[%d]: got %v, want %v", i, output.RecvIv()[i], input.RecvIv()[i])
				}
			}
		})
	}
}

package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestStartErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := StartError{
				length: 4,
				bytes:  []byte{0x01, 0x02, 0x03, 0x04},
			}
			output := StartError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Length() != input.Length() {
				t.Errorf("length: got %v, want %v", output.Length(), input.Length())
			}
			if !bytes.Equal(output.Bytes(), input.Bytes()) {
				t.Errorf("bytes: got %v, want %v", output.Bytes(), input.Bytes())
			}
		})
	}
}

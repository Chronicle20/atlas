package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=socket/serverbound/StartError version=gms_v83 ida=0x494ed1
// packet-audit:verify packet=socket/serverbound/StartError version=gms_v87 ida=0x4a6e5a
// packet-audit:verify packet=socket/serverbound/StartError version=gms_v95 ida=0x4aef10
// packet-audit:verify packet=socket/serverbound/StartError version=jms_v185 ida=0x4b0066
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

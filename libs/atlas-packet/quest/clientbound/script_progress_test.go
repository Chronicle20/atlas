package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestScriptProgress(t *testing.T) {
	input := NewScriptProgress("quest progress message")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestScriptProgressWireShape verifies the wire layout against
// CWvsContext::OnScriptProgressMessage (GMS v95 @ 0x9e5110):
//
//	DecodeStr → length-prefixed ASCII string (uint16 length + bytes)
//
// Atlas WriteAsciiString produces the same encoding. All versions identical.
func TestScriptProgressWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	msg := "hello"
	in := NewScriptProgress(msg)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			// 2 (length uint16 LE) + len(msg) bytes
			want := 2 + len(msg)
			if len(b) != want {
				t.Fatalf("wire size = %d bytes, want %d: % x", len(b), want, b)
			}
			gotLen := int(b[0]) | int(b[1])<<8
			if gotLen != len(msg) {
				t.Errorf("string length prefix = %d, want %d", gotLen, len(msg))
			}
			if string(b[2:]) != msg {
				t.Errorf("string payload = %q, want %q", string(b[2:]), msg)
			}
		})
	}
}

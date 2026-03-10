package field

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestSetFieldEncode(t *testing.T) {
	input := NewSetField(channel.Id(1), []byte{0x01, 0x02, 0x03})
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			encoded := input.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Error("expected non-empty encoded bytes")
			}
		})
	}
}

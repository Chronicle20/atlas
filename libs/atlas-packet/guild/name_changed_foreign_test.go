package guild

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestForeignNameChangedRoundTrip(t *testing.T) {
	input := NewForeignNameChanged(1001, "NewGuildName")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ForeignNameChanged{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

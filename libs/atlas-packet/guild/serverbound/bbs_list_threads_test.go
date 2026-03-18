package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestBBSListThreadsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSListThreads{startIndex: 10}
			output := BBSListThreads{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.StartIndex() != input.StartIndex() {
				t.Errorf("startIndex: got %v, want %v", output.StartIndex(), input.StartIndex())
			}
		})
	}
}

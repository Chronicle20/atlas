package chat

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestMultiRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Multi{chatType: 1, recipients: []uint32{100, 200, 300}, chatText: "party chat"}
			output := Multi{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ChatType() != input.ChatType() {
				t.Errorf("chatType: got %v, want %v", output.ChatType(), input.ChatType())
			}
			if len(output.Recipients()) != len(input.Recipients()) {
				t.Fatalf("recipients length: got %v, want %v", len(output.Recipients()), len(input.Recipients()))
			}
			for i, r := range output.Recipients() {
				if r != input.Recipients()[i] {
					t.Errorf("recipients[%d]: got %v, want %v", i, r, input.Recipients()[i])
				}
			}
			if output.ChatText() != input.ChatText() {
				t.Errorf("chatText: got %v, want %v", output.ChatText(), input.ChatText())
			}
		})
	}
}

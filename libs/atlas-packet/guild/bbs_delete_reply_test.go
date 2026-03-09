package guild

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestBBSDeleteReplyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSDeleteReply{threadId: 3, replyId: 12}
			output := BBSDeleteReply{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ThreadId() != input.ThreadId() {
				t.Errorf("threadId: got %v, want %v", output.ThreadId(), input.ThreadId())
			}
			if output.ReplyId() != input.ReplyId() {
				t.Errorf("replyId: got %v, want %v", output.ReplyId(), input.ReplyId())
			}
		})
	}
}

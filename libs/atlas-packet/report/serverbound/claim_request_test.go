package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func TestClaimRequestChatClaimGolden(t *testing.T) {
	// bChatClaim=1 appends the client-supplied chat log string.
	input := NewClaimRequest(1, "bob", 0x02, "hi", "yo")
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{
		0x01,                         // bChatClaim
		0x03, 0x00, 0x62, 0x6F, 0x62, // "bob"
		0x02,                   // nType
		0x02, 0x00, 0x68, 0x69, // "hi"
		0x02, 0x00, 0x79, 0x6F, // "yo"
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestClaimRequestRegularGolden(t *testing.T) {
	// bChatClaim=0: no chat log trailer.
	input := NewClaimRequest(0, "bob", 0x05, "hi", "")
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{
		0x00,
		0x03, 0x00, 0x62, 0x6F, 0x62,
		0x05,
		0x02, 0x00, 0x68, 0x69,
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}

	// Verify decode direction: bChatClaim=0 branch must NOT read a trailing chatLog.
	l, _ := testlog.NewNullLogger()
	output := ClaimRequest{}
	req := request.Request(expected)
	reader := request.NewRequestReader(&req, 0)
	output.Decode(l, ctx)(&reader, nil)

	if reader.Available() > 0 {
		t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
	}
	if output.IsChatClaim() != input.IsChatClaim() || output.TargetName() != input.TargetName() ||
		output.ReasonType() != input.ReasonType() || output.Description() != input.Description() ||
		output.ChatLog() != input.ChatLog() {
		t.Errorf("decode mismatch: got %+v want %+v", output, input)
	}
}

func TestClaimRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewClaimRequest(1, "alice", 0x03, "harassment in fm1", "alice: mean words")
			output := ClaimRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IsChatClaim() != input.IsChatClaim() || output.TargetName() != input.TargetName() ||
				output.ReasonType() != input.ReasonType() || output.Description() != input.Description() ||
				output.ChatLog() != input.ChatLog() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}

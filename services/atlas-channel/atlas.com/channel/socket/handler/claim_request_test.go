package handler

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	reportsb "github.com/Chronicle20/atlas/libs/atlas-packet/report/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestClaimRequestDecodeRegular pins the wire format ClaimRequestHandleFunc
// consumes for a regular (non-chat) claim: bChatClaim(1)=0, no trailing
// chatLog string.
func TestClaimRequestDecodeRegular(t *testing.T) {
	raw := []byte{
		0x00,                      // bChatClaim = 0
		0x03, 0x00, 'b', 'o', 'b', // sTargetCharacterName = "bob"
		0x03,                                                         // nType = 3
		0x0A, 0x00, 'h', 'a', 'r', 'a', 's', 's', 'm', 'e', 'n', 't', // sContext = "harassment"
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := reportsb.ClaimRequest{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.IsChatClaim() {
		t.Errorf("chatClaim: got true, want false")
	}
	if p.TargetName() != "bob" {
		t.Errorf("targetName: got %q, want %q", p.TargetName(), "bob")
	}
	if p.ReasonType() != 3 {
		t.Errorf("reasonType: got %d, want 3", p.ReasonType())
	}
	if p.Description() != "harassment" {
		t.Errorf("description: got %q, want %q", p.Description(), "harassment")
	}
	if p.ChatLog() != "" {
		t.Errorf("chatLog: got %q, want empty", p.ChatLog())
	}
	if p.Operation() != reportsb.ClaimRequestHandle {
		t.Errorf("operation: got %q, want %q", p.Operation(), reportsb.ClaimRequestHandle)
	}
}

// TestClaimRequestDecodeChatClaim pins the wire format for a chat/harassment
// claim: bChatClaim(1)=1, followed by the trailing chatLog string.
func TestClaimRequestDecodeChatClaim(t *testing.T) {
	raw := []byte{
		0x01,                                // bChatClaim = 1
		0x05, 0x00, 'a', 'l', 'i', 'c', 'e', // sTargetCharacterName = "alice"
		0x02,                           // nType = 2
		0x04, 0x00, 's', 'p', 'a', 'm', // sContext = "spam"
		0x08, 0x00, 'a', 'l', 'i', 'c', 'e', ':', ' ', 'x', // chatLog = "alice: x"
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := reportsb.ClaimRequest{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if !p.IsChatClaim() {
		t.Errorf("chatClaim: got false, want true")
	}
	if p.TargetName() != "alice" {
		t.Errorf("targetName: got %q, want %q", p.TargetName(), "alice")
	}
	if p.ReasonType() != 2 {
		t.Errorf("reasonType: got %d, want 2", p.ReasonType())
	}
	if p.Description() != "spam" {
		t.Errorf("description: got %q, want %q", p.Description(), "spam")
	}
	if p.ChatLog() != "alice: x" {
		t.Errorf("chatLog: got %q, want %q", p.ChatLog(), "alice: x")
	}
}

// TestClaimRequestHandleFuncSymbol verifies the handler constructor returns a
// non-nil closure with the standard handler signature.
func TestClaimRequestHandleFuncSymbol(t *testing.T) {
	got := ClaimRequestHandleFunc(logrus.New(), context.Background(), nil)
	if got == nil {
		t.Fatal("ClaimRequestHandleFunc returned nil closure")
	}
}

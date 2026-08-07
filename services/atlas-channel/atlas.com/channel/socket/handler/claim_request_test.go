package handler

import (
	"atlas-channel/character"
	"atlas-channel/report"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"io"
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

// recordingClaimDeps captures what submitClaim did without a REST client,
// a Kafka producer or a session registry.
type recordingClaimDeps struct {
	meso      uint32
	lookupErr error
	submitted int
	notices   []writer.ClaimResultCode
}

func (d *recordingClaimDeps) deps() claimSubmitDeps {
	return claimSubmitDeps{
		getCharacter: func(characterId uint32) (character.Model, error) {
			if d.lookupErr != nil {
				return character.Model{}, d.lookupErr
			}
			return character.NewModelBuilder().SetId(characterId).SetMeso(d.meso).Build()
		},
		submitClaim: func(_ reportsb.ClaimRequest) error { d.submitted++; return nil },
		notice:      func(code writer.ClaimResultCode) { d.notices = append(d.notices, code) },
	}
}

func TestSubmitClaimChargesOnlyWhenAffordable(t *testing.T) {
	tests := []struct {
		name          string
		meso          uint32
		wantSubmitted int
		wantNotices   []writer.ClaimResultCode
	}{
		{
			name:          "exactly the fee submits",
			meso:          uint32(report.ClaimCostMesos),
			wantSubmitted: 1,
		},
		{
			name:        "one meso short is refused with the client's meso notice",
			meso:        uint32(report.ClaimCostMesos) - 1,
			wantNotices: []writer.ClaimResultCode{writer.ClaimResultNotEnoughMesos},
		},
		{
			name:        "broke is refused",
			meso:        0,
			wantNotices: []writer.ClaimResultCode{writer.ClaimResultNotEnoughMesos},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recordingClaimDeps{meso: tt.meso}
			submitClaim(nullHandlerLogger(), 77, reportsb.ClaimRequest{}, rec.deps())

			if rec.submitted != tt.wantSubmitted {
				t.Errorf("submitted: got %d, want %d", rec.submitted, tt.wantSubmitted)
			}
			if len(rec.notices) != len(tt.wantNotices) {
				t.Fatalf("notices: got %v, want %v", rec.notices, tt.wantNotices)
			}
			for i, want := range tt.wantNotices {
				if rec.notices[i] != want {
					t.Errorf("notice %d: got %s, want %s", i, rec.notices[i], want)
				}
			}
		})
	}
}

// An unresolvable reporter must not silently drop the claim: the reporter
// gets TRY_AGAIN, and no command is emitted for a character we could not read.
func TestSubmitClaimUnresolvableReporterNoticesTryAgain(t *testing.T) {
	rec := &recordingClaimDeps{meso: 100000, lookupErr: errors.New("character service down")}
	submitClaim(nullHandlerLogger(), 77, reportsb.ClaimRequest{}, rec.deps())

	if rec.submitted != 0 {
		t.Errorf("submitted: got %d, want 0", rec.submitted)
	}
	if len(rec.notices) != 1 || rec.notices[0] != writer.ClaimResultTryAgain {
		t.Errorf("notices: got %v, want [%s]", rec.notices, writer.ClaimResultTryAgain)
	}
}

func nullHandlerLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

package handler

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
)

// gms83MessageType mirrors the "messageType" table the NPCContinueConversation
// handler is configured with (template_gms_83_1.json). The handler reverse-maps
// the client's lastMessageType byte through this table, so the test drives the
// real config rather than hardcoded bytes.
var gms83MessageType = map[string]interface{}{
	"SAY":                    float64(0),
	"ASK_YES_NO":             float64(1),
	"ASK_TEXT":               float64(2),
	"ASK_NUMBER":             float64(3),
	"ASK_MENU":               float64(4),
	"ASK_QUIZ":               float64(5),
	"ASK_SPEED_QUIZ":         float64(6),
	"ASK_AVATAR":             float64(7),
	"ASK_MEMBER_SHOP_AVATAR": float64(8),
	"ASK_PET":                float64(9),
	"ASK_PET_ALL":            float64(10),
	"ASK_YES_NO_QUEST":       float64(12),
	"ASK_BOX_TEXT":           float64(13),
	"ASK_SLIDE_MENU":         float64(14),
}

// TestContinueConversationBodyKind pins the discriminator that decides which
// trailing body the serverbound continue-conversation packet carries. The
// version-specific byte numbering comes from tenant config; only the
// name→body-kind grouping is asserted here.
//
//	SAY / ASK_YES_NO / ASK_YES_NO_QUEST              → no trailing body
//	ASK_TEXT / ASK_BOX_TEXT                          → text reply
//	ASK_NUMBER / ASK_MENU / ASK_AVATAR / ASK_SLIDE_MENU → selection reply
func TestContinueConversationBodyKind(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	opts := map[string]interface{}{"messageType": gms83MessageType}

	cases := []struct {
		name    string
		msgType byte
		want    bodyKind
	}{
		{"SAY", 0, bodyNone},
		{"ASK_YES_NO", 1, bodyNone},
		{"ASK_YES_NO_QUEST", 12, bodyNone},
		{"ASK_TEXT", 2, bodyText},
		{"ASK_BOX_TEXT", 13, bodyText},
		{"ASK_NUMBER", 3, bodySelection},
		{"ASK_MENU", 4, bodySelection},
		{"ASK_AVATAR", 7, bodySelection},
		{"ASK_SLIDE_MENU", 14, bodySelection},
	}
	for _, c := range cases {
		if got := bodyKindFor(l, opts, c.msgType); got != c.want {
			t.Errorf("%s (byte %d): got %v, want %v", c.name, c.msgType, got, c.want)
		}
	}
}

// TestContinueConversationBodyKindUnconfigured guards the failure mode that
// caused the original regression: when the byte is not present in the
// messageType table (e.g. missing handler config), the handler must fall back
// to bodyNone rather than mis-parse the trailing bytes.
func TestContinueConversationBodyKindUnconfigured(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	// Empty options: nothing configured.
	if got := bodyKindFor(l, map[string]interface{}{}, 4); got != bodyNone {
		t.Errorf("missing messageType config: got %v, want bodyNone", got)
	}

	// Byte not assigned to any known name in the table.
	opts := map[string]interface{}{"messageType": gms83MessageType}
	if got := bodyKindFor(l, opts, 99); got != bodyNone {
		t.Errorf("unknown byte 99: got %v, want bodyNone", got)
	}
}

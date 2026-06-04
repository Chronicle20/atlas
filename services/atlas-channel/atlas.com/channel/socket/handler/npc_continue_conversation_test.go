package handler

import "testing"

// TestContinueConversationBodyKind pins the discriminator that decides which
// trailing body the serverbound continue-conversation packet carries, keyed on
// the client's lastMessageType (task-080 B2.1).
//
//	0/1/2/13 (Say/AskYesNo)              → no trailing body
//	3 (OnAskText) / 14 (OnAskBoxText)    → text reply
//	5 (OnAskMenu) / 8 (OnAskAvatar) / 9  → selection
func TestContinueConversationBodyKind(t *testing.T) {
	cases := []struct {
		msgType byte
		want    bodyKind
	}{
		{0, bodyNone}, {1, bodyNone}, {2, bodyNone}, {13, bodyNone},
		{3, bodyText}, {14, bodyText},
		{5, bodySelection}, {8, bodySelection}, {9, bodySelection},
	}
	for _, c := range cases {
		if got := bodyKindFor(c.msgType); got != c.want {
			t.Errorf("msgType %d: got %v, want %v", c.msgType, got, c.want)
		}
	}
}

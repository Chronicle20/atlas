package consumable

import "testing"

// TestSolomonErrorWireValues pins the wire values atlas-channel hand-mirrors
// in its own kafka/message/consumable/kafka.go (see that file's comment). A
// typo on either side silently routes a Solomon rejection to the
// CONSUME_FAILED catch-all instead of its distinct message.
func TestSolomonErrorWireValues(t *testing.T) {
	tests := []struct {
		got  string
		want string
	}{
		{ErrorTypeSolomonNoExperience, "SOLOMON_NO_EXPERIENCE"},
		{ErrorTypeSolomonLevelExceeded, "SOLOMON_LEVEL_EXCEEDED"},
		{ErrorTypeSolomonBalanceNotEmpty, "SOLOMON_BALANCE_NOT_EMPTY"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

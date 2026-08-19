package character

import "testing"

func TestPresenceStateValues(t *testing.T) {
	// These literals cross the atlas-maps -> atlas-channel REST boundary.
	// Renaming or re-casing one silently degrades /find to "not findable".
	cases := []struct {
		state PresenceState
		want  string
	}{
		{PresenceStateOffline, "OFFLINE"},
		{PresenceStateInField, "IN_FIELD"},
		{PresenceStateInCashShop, "IN_CASH_SHOP"},
	}
	for _, c := range cases {
		if string(c.state) != c.want {
			t.Errorf("state = %q, want %q", string(c.state), c.want)
		}
	}
}

func TestParsePresenceState(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  PresenceState
	}{
		{"offline", "OFFLINE", PresenceStateOffline},
		{"in field", "IN_FIELD", PresenceStateInField},
		{"in cash shop", "IN_CASH_SHOP", PresenceStateInCashShop},
		// The zero value is OFFLINE by design: a row written before this
		// column existed, or a REST payload from an atlas-maps that has not
		// been redeployed, must fail toward "not findable" rather than
		// asserting liveness.
		{"empty string is offline", "", PresenceStateOffline},
		{"unrecognised is offline", "IN_ORBIT", PresenceStateOffline},
		{"lowercase is not accepted", "in_field", PresenceStateOffline},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParsePresenceState(c.input); got != c.want {
				t.Errorf("ParsePresenceState(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

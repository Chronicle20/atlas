package handler

import "testing"

func TestViciousHammerTokenRoundTrip(t *testing.T) {
	cases := []struct {
		name       string
		hammerSlot int16
		equipSlot  int16
	}{
		{"inventory target", 5, 3},
		{"equipped target (negative slot)", 1, -5},
		{"high slots", 96, 24},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := packViciousHammerToken(tc.hammerSlot, tc.equipSlot)
			h, e := unpackViciousHammerToken(token)
			if h != tc.hammerSlot || e != tc.equipSlot {
				t.Errorf("got (%d, %d), want (%d, %d)", h, e, tc.hammerSlot, tc.equipSlot)
			}
		})
	}
}

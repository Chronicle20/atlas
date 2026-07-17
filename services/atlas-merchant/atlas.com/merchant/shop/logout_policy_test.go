package shop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Logout policy (owner decision 2026-07-14, merchant-lifecycle-audit Q5):
//   - a Draft shop of EITHER type is an owner-attached setup session -> close.
//   - personal shops close in every live state (they cannot outlive the owner).
//   - an Open hired merchant survives logout (runs owner-detached until
//     expiry/explicit close).
//   - a hired merchant caught in Maintenance reverts to running via
//     exit-maintenance (leaving it in Maintenance would strand it
//     unenterable forever).
func TestLogoutAction(t *testing.T) {
	cases := []struct {
		name     string
		shopType ShopType
		state    State
		want     LogoutOutcome
	}{
		{"personal draft", CharacterShop, Draft, LogoutClose},
		{"personal open", CharacterShop, Open, LogoutClose},
		{"personal maintenance", CharacterShop, Maintenance, LogoutClose},
		{"personal closed", CharacterShop, Closed, LogoutNone},
		{"merchant draft", HiredMerchant, Draft, LogoutClose},
		{"merchant open", HiredMerchant, Open, LogoutNone},
		{"merchant maintenance", HiredMerchant, Maintenance, LogoutExitMaintenance},
		{"merchant closed", HiredMerchant, Closed, LogoutNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, LogoutAction(tc.shopType, tc.state))
		})
	}
}

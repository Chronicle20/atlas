package shopscanner

import (
	"testing"

	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
	"github.com/stretchr/testify/require"
)

// valid returns a WarpCheck that passes every rung — each case below breaks
// exactly one rung and expects the design §4.2 code.
func valid() WarpCheck {
	return WarpCheck{
		HasSearch:        true,
		OwnerId:          30001,
		CharacterId:      1,
		CharacterHp:      50,
		CurrentMapFM:     true,
		ShopFound:        true,
		ShopWorldId:      0,
		SessionWorldId:   0,
		ShopChannelId:    1,
		SessionChannelId: 1,
		ShopMapId:        910000004,
		EchoedMapId:      910000004,
		ShopState:        2, // Open
		ListingPresent:   true,
	}
}

func TestEvaluateWarp(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*WarpCheck)
		code   merchantpkt.ShopLinkResultCode
		ok     bool
	}{
		{"all valid", func(c *WarpCheck) {}, "", true},
		{"outside FM", func(c *WarpCheck) { c.CurrentMapFM = false }, merchantpkt.ShopLinkResultCodeFMOnly, false},
		{"no prior search", func(c *WarpCheck) { c.HasSearch = false }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"own shop", func(c *WarpCheck) { c.OwnerId = 1 }, merchantpkt.ShopLinkResultCodeDenied, false},
		{"dead", func(c *WarpCheck) { c.CharacterHp = 0 }, merchantpkt.ShopLinkResultCodeDead, false},
		{"shop missing", func(c *WarpCheck) { c.ShopFound = false }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"wrong world", func(c *WarpCheck) { c.ShopWorldId = 1 }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"echo tamper (map mismatch)", func(c *WarpCheck) { c.EchoedMapId = 910000005 }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"shop outside FM", func(c *WarpCheck) { c.ShopMapId = 100000000; c.EchoedMapId = 100000000 }, merchantpkt.ShopLinkResultCodeFMOnly, false},
		{"cross channel", func(c *WarpCheck) { c.ShopChannelId = 2 }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"maintenance", func(c *WarpCheck) { c.ShopState = 3 }, merchantpkt.ShopLinkResultCodeMaintenance, false},
		{"closed state", func(c *WarpCheck) { c.ShopState = 4 }, merchantpkt.ShopLinkResultCodeClosed, false},
		{"listing gone", func(c *WarpCheck) { c.ListingPresent = false }, merchantpkt.ShopLinkResultCodeBusy, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := valid()
			tc.mutate(&c)
			code, ok := EvaluateWarp(c)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.code, code)
		})
	}
}

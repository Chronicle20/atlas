package shopscanner

import (
	"atlas-channel/merchant"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
)

// WarpCheck is the pre-fetched state the OWL_WARP validation ladder
// (design §4.2) evaluates. The handler gathers; this decides.
type WarpCheck struct {
	HasSearch        bool
	OwnerId          uint32
	CharacterId      uint32
	CharacterHp      uint16
	CurrentMapFM     bool
	ShopFound        bool
	ShopWorldId      world.Id
	SessionWorldId   world.Id
	ShopChannelId    channel.Id
	SessionChannelId channel.Id
	ShopMapId        uint32
	EchoedMapId      uint32
	ShopState        byte
	ListingPresent   bool
}

// EvaluateWarp walks the ladder in order; the first failing rung yields its
// SHOP_LINK code. ("", true) means the warp may proceed.
func EvaluateWarp(c WarpCheck) (merchantpkt.ShopLinkResultCode, bool) {
	if !c.CurrentMapFM {
		return merchantpkt.ShopLinkResultCodeFMOnly, false
	}
	if !c.HasSearch {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if c.OwnerId == c.CharacterId {
		return merchantpkt.ShopLinkResultCodeDenied, false
	}
	if c.CharacterHp == 0 {
		return merchantpkt.ShopLinkResultCodeDead, false
	}
	if !c.ShopFound {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if c.ShopWorldId != c.SessionWorldId {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if c.ShopMapId != c.EchoedMapId {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if !_map.IsFreeMarketRoom(_map.Id(c.ShopMapId)) {
		return merchantpkt.ShopLinkResultCodeFMOnly, false
	}
	if c.ShopChannelId != c.SessionChannelId {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if c.ShopState == merchant.StateMaintenance {
		return merchantpkt.ShopLinkResultCodeMaintenance, false
	}
	if c.ShopState != merchant.StateOpen {
		return merchantpkt.ShopLinkResultCodeClosed, false
	}
	if !c.ListingPresent {
		return merchantpkt.ShopLinkResultCodeBusy, false
	}
	return "", true
}

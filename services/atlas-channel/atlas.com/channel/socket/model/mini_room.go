package model

import (
	"atlas-channel/character"

	"github.com/Chronicle20/atlas-constants/item"
)

type MiniRoomType byte

const (
	OmokMiniRoom         MiniRoomType = 1 // COmokDlg
	MatchCardMiniRoom    MiniRoomType = 2 // CMemoryGameDlg
	TradeMiniRoom        MiniRoomType = 3 // CTradingRoomDlg
	PersonalShopMiniRoom MiniRoomType = 4 // CPersonalShopDlg
	MerchantShopMiniRoom MiniRoomType = 5 // CEntrustedShopDlg
	CashTradeMiniRoom    MiniRoomType = 6 // CCashTradingRoomDlg
)

type MiniRoom struct {
	ItemId   item.Id
	Type     MiniRoomType
	MaxUsers byte
	Visitors []MiniRoomVisitor
}

func (m MiniRoom) IsMerchant() bool {
	return m.Type == MerchantShopMiniRoom
}

type MiniRoomVisitor struct {
	Slot      byte
	Character character.Model
}

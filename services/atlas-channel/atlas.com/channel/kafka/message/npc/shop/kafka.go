// Package shops mirrors atlas-npc-shops' COMMAND_TOPIC_NPC_SHOP /
// EVENT_TOPIC_NPC_SHOP_STATUS contract. atlas-npc-shops owns it; see
// tools/npc-shop-contract-mirror-guard.sh.
package shops

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_NPC_SHOP"
)

const (
	CommandShopEnter    = "ENTER"
	CommandShopExit     = "EXIT"
	CommandShopBuy      = "BUY"
	CommandShopSell     = "SELL"
	CommandShopRecharge = "RECHARGE"
)

type Command[E any] struct {
	// TransactionId correlates a command with the saga step that issued it.
	// uuid.Nil for the ordinary NPC-talk path, which has no saga.
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CommandShopEnterBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
}

type CommandShopExitBody struct{}

type CommandShopBuyBody struct {
	Slot           uint16 `json:"slot"`
	ItemTemplateId uint32 `json:"itemTemplateId"`
	Quantity       uint32 `json:"quantity"`
	DiscountPrice  uint32 `json:"discountPrice"`
}

type CommandShopSellBody struct {
	Slot           int16  `json:"slot"`
	ItemTemplateId uint32 `json:"itemTemplateId"`
	Quantity       uint32 `json:"quantity"`
}

type CommandShopRechargeBody struct {
	Slot uint16 `json:"slot"`
}

const (
	EnvStatusEventTopic topic.Token = "EVENT_TOPIC_NPC_SHOP_STATUS"
)

const (
	StatusEventTypeEntered = "ENTERED"
	StatusEventTypeExited  = "EXITED"
	StatusEventTypeError   = "ERROR"

	// StatusEventTypeEnterError reports that an ENTER command failed. It is
	// deliberately NOT StatusEventTypeError: the channel renders that one as a
	// CONFIRM_SHOP_TRANSACTION packet, and CShopDlg::OnPacket @0x756da7 throws
	// CDisconnectException when that packet arrives with no buy/sell/recharge
	// request outstanding. An enter failure has none.
	StatusEventTypeEnterError = "ENTER_ERROR"

	// Reasons carried by StatusEventEnterErrorBody.
	EnterErrorShopNotFound  = "SHOP_NOT_FOUND"
	EnterErrorAlreadyInShop = "ALREADY_IN_SHOP"

	ErrorOk                     = "OK"
	ErrorOutOfStock             = "OUT_OF_STOCK"
	ErrorNotEnoughMoney         = "NOT_ENOUGH_MONEY"
	ErrorInventoryFull          = "INVENTORY_FULL"
	ErrorOutOfStock2            = "OUT_OF_STOCK_2"
	ErrorOutOfStock3            = "OUT_OF_STOCK_3"
	ErrorNotEnoughMoney2        = "NOT_ENOUGH_MONEY_2"
	ErrorNeedMoreItems          = "NEED_MORE_ITEMS"
	ErrorOverLevelRequirement   = "OVER_LEVEL_REQUIREMENT"
	ErrorUnderLevelRequirement  = "UNDER_LEVEL_REQUIREMENT"
	ErrorTradeLimit             = "TRADE_LIMIT"
	ErrorGenericError           = "GENERIC_ERROR"
	ErrorGenericErrorWithReason = "GENERIC_ERROR_WITH_REASON"
)

type StatusEvent[E any] struct {
	// TransactionId echoes the originating command's id so a saga can accept
	// the event. uuid.Nil when the command carried none.
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventEnteredBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
}

type StatusEventExitedBody struct{}

type StatusEventErrorBody struct {
	Error      string `json:"error"`
	LevelLimit uint32 `json:"levelLimit"`
	Reason     string `json:"reason"`
}

type StatusEventEnterErrorBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
	Reason        string `json:"reason"`
}

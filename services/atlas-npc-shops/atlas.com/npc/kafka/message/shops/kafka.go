package shops

const (
	EnvCommandTopic     = "COMMAND_TOPIC_NPC_SHOP"
	CommandShopEnter    = "ENTER"
	CommandShopExit     = "EXIT"
	CommandShopBuy      = "BUY"
	CommandShopSell     = "SELL"
	CommandShopRecharge = "RECHARGE"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CommandShopEnterBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
}

type CommandShopExitBody struct {
}

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
	EnvStatusEventTopic    = "EVENT_TOPIC_NPC_SHOP_STATUS"
	StatusEventTypeEntered = "ENTERED"
	StatusEventTypeExited  = "EXITED"
	StatusEventTypeError   = "ERROR"

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
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventEnteredBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
}

type StatusEventExitedBody struct {
}

type StatusEventErrorBody struct {
	Error      string `json:"error"`
	LevelLimit uint32 `json:"levelLimit"`
	Reason     string `json:"reason"`
}

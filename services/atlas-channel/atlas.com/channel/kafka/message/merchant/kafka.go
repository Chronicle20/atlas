package merchant

import (
	"atlas-channel/kafka/message/asset"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_MERCHANT"

	CommandPlaceShop        = "PLACE_SHOP"
	CommandOpenShop         = "OPEN_SHOP"
	CommandCloseShop        = "CLOSE_SHOP"
	CommandEnterMaintenance = "ENTER_MAINTENANCE"
	CommandExitMaintenance  = "EXIT_MAINTENANCE"
	CommandEnterShop        = "ENTER_SHOP"
	CommandExitShop         = "EXIT_SHOP"
	CommandSendMessage      = "SEND_MESSAGE"
	CommandAddListing       = "ADD_LISTING"
	CommandRemoveListing    = "REMOVE_LISTING"
	CommandUpdateListing    = "UPDATE_LISTING"
	CommandPurchaseBundle   = "PURCHASE_BUNDLE"
	CommandRecordItemSearch = "RECORD_ITEM_SEARCH"
	CommandWithdrawMeso     = "WITHDRAW_MESO"
	CommandOrganizeListings = "ORGANIZE_LISTINGS"
	CommandAddBlacklist     = "ADD_BLACKLIST"
	CommandRemoveBlacklist  = "REMOVE_BLACKLIST"
)

type Command[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

type CommandPlaceShopBody struct {
	ShopType     byte      `json:"shopType"`
	Title        string    `json:"title"`
	MapId        uint32    `json:"mapId"`
	InstanceId   uuid.UUID `json:"instanceId"`
	X            int16     `json:"x"`
	Y            int16     `json:"y"`
	PermitItemId uint32    `json:"permitItemId"`
}

type CommandOpenShopBody struct {
	ShopId string `json:"shopId"`
}

type CommandCloseShopBody struct {
	ShopId string `json:"shopId"`
}

type CommandBlacklistBody struct {
	ShopId string `json:"shopId"`
	Name   string `json:"name"`
	// BannedCharacterId, when non-zero, is a visitor to eject (USER_BANNED) as
	// part of the ban. Zero for a name-only blacklist add.
	BannedCharacterId uint32 `json:"bannedCharacterId,omitempty"`
}

type CommandEnterShopBody struct {
	VisitorName string `json:"visitorName"`
	ShopId      string `json:"shopId"`
}

type CommandExitShopBody struct {
	ShopId string `json:"shopId"`
}

type CommandSendMessageBody struct {
	ShopId  string `json:"shopId"`
	Content string `json:"content"`
}

type CommandEnterMaintenanceBody struct {
	ShopId string `json:"shopId"`
}

type CommandExitMaintenanceBody struct {
	ShopId string `json:"shopId"`
}

type CommandAddListingBody struct {
	ShopId         string          `json:"shopId"`
	ItemId         uint32          `json:"itemId"`
	ItemType       byte            `json:"itemType"`
	BundleSize     uint16          `json:"bundleSize"`
	BundleCount    uint16          `json:"bundleCount"`
	PricePerBundle uint32          `json:"pricePerBundle"`
	Slot           int16           `json:"slot"`
	InventoryType  byte            `json:"inventoryType"`
	AssetId        uint32          `json:"assetId"`
	ItemSnapshot   asset.AssetData `json:"itemSnapshot"`
}

type CommandRemoveListingBody struct {
	ShopId       string `json:"shopId"`
	ListingIndex uint16 `json:"listingIndex"`
}

type CommandUpdateListingBody struct {
	ShopId         string `json:"shopId"`
	ListingIndex   uint16 `json:"listingIndex"`
	PricePerBundle uint32 `json:"pricePerBundle"`
	BundleSize     uint16 `json:"bundleSize"`
	BundleCount    uint16 `json:"bundleCount"`
}

type CommandPurchaseBundleBody struct {
	ShopId       string `json:"shopId"`
	ListingIndex uint16 `json:"listingIndex"`
	BundleCount  uint16 `json:"bundleCount"`
}

type CommandWithdrawMesoBody struct {
	ShopId string `json:"shopId"`
}

type CommandOrganizeListingsBody struct {
	ShopId string `json:"shopId"`
}

type CommandRecordItemSearchBody struct {
	ItemId uint32 `json:"itemId"`
}

const (
	EnvStatusEventTopic = "EVENT_TOPIC_MERCHANT_STATUS"

	StatusEventShopOpened            = "SHOP_OPENED"
	StatusEventShopSetup             = "SHOP_SETUP"
	StatusEventShopClosed            = "SHOP_CLOSED"
	StatusEventMaintenanceEntered    = "MAINTENANCE_ENTERED"
	StatusEventMaintenanceExited     = "MAINTENANCE_EXITED"
	StatusEventVisitorEntered        = "VISITOR_ENTERED"
	StatusEventVisitorExited         = "VISITOR_EXITED"
	StatusEventVisitorEjected        = "VISITOR_EJECTED"
	StatusEventCapacityFull          = "CAPACITY_FULL"
	StatusEventPurchaseFailed        = "PURCHASE_FAILED"
	StatusEventFrederickNotification = "FREDERICK_NOTIFICATION"
	StatusEventMessageSent           = "MESSAGE_SENT"
	StatusEventShopCreateFailed      = "SHOP_CREATE_FAILED"
	StatusEventShopUpdated           = "SHOP_UPDATED"
	StatusEventEnterFailed           = "ENTER_FAILED"
	StatusEventBlacklistUpdated      = "BLACKLIST_UPDATED"

	EnterFailReasonUndergoingMaintenance = "UNDERGOING_MAINTENANCE"
	EnterFailReasonRoomClosed            = "ROOM_CLOSED"
	EnterFailReasonBlacklisted           = "BLACKLISTED"

	// Reasons carried by StatusEventShopCreateFailedBody (mirror of atlas-merchant).
	ShopCreateFailReasonTooCloseToPortal = "TOO_CLOSE_TO_PORTAL"
	ShopCreateFailReasonTooCloseToShop   = "TOO_CLOSE_TO_SHOP"
	ShopCreateFailReasonNotFreeMarket    = "NOT_FREE_MARKET"
	ShopCreateFailReasonUnable           = "UNABLE"

	EnvListingEventTopic = "EVENT_TOPIC_MERCHANT_LISTING"

	ListingEventPurchased = "LISTING_PURCHASED"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventShopOpenedBody struct {
	ShopId     string     `json:"shopId"`
	ShopType   byte       `json:"shopType"`
	WorldId    world.Id   `json:"worldId"`
	ChannelId  channel.Id `json:"channelId"`
	MapId      uint32     `json:"mapId"`
	InstanceId uuid.UUID  `json:"instanceId"`
	Title      string     `json:"title"`
	X          int16      `json:"x"`
	Y          int16      `json:"y"`
}

type StatusEventShopClosedBody struct {
	ShopId      string `json:"shopId"`
	CloseReason byte   `json:"closeReason"`
}

type StatusEventVisitorBody struct {
	ShopId      string `json:"shopId"`
	CharacterId uint32 `json:"characterId"`
	Slot        byte   `json:"slot"`
	// LeaveReason is the client "leaveReason" table key sent to an ejected
	// visitor (VISITOR_EJECTED) so their room UI shows the right message
	// instead of an empty dialog. Empty for enter/leave events.
	LeaveReason string `json:"leaveReason,omitempty"`
}

type StatusEventCapacityFullBody struct {
	ShopId string `json:"shopId"`
}

type StatusEventShopCreateFailedBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	Reason    string     `json:"reason"`
}

type StatusEventPurchaseFailedBody struct {
	ShopId string `json:"shopId"`
	Reason string `json:"reason"`
}

type StatusEventFrederickNotificationBody struct {
	DaysSinceStorage uint16 `json:"daysSinceStorage"`
}

type StatusEventMessageSentBody struct {
	ShopId      string `json:"shopId"`
	CharacterId uint32 `json:"characterId"`
	Slot        byte   `json:"slot"`
	Content     string `json:"content"`
}

type StatusEventShopUpdatedBody struct {
	ShopId string `json:"shopId"`
}

type StatusEventBlacklistUpdatedBody struct {
	ShopId string `json:"shopId"`
}

type StatusEventEnterFailedBody struct {
	ShopId string `json:"shopId"`
	Reason string `json:"reason"`
}

type ListingEvent[E any] struct {
	ShopId string `json:"shopId"`
	Type   string `json:"type"`
	Body   E      `json:"body"`
}

type ListingEventPurchasedBody struct {
	ListingIndex     uint16 `json:"listingIndex"`
	BuyerCharacterId uint32 `json:"buyerCharacterId"`
	BundleCount      uint16 `json:"bundleCount"`
	BundlesRemaining uint16 `json:"bundlesRemaining"`
}

package compartment

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopicCompartmentTransfer = "COMMAND_TOPIC_COMPARTMENT_TRANSFER"
	InventoryTypeCharacter             = "CHARACTER"
	InventoryTypeCashShop              = "CASH_SHOP"
)

type TransferCommand struct {
	TransactionId       uuid.UUID `json:"transactionId"`
	AccountId           uint32    `json:"accountId"`
	CharacterId         uint32    `json:"characterId"`
	AssetId             uint32    `json:"assetId"`
	FromCompartmentId   uuid.UUID `json:"fromCompartmentId"`
	FromCompartmentType byte      `json:"fromCompartmentType"`
	FromInventoryType   string    `json:"fromInventoryType"`
	ToCompartmentId     uuid.UUID `json:"toCompartmentId"`
	ToCompartmentType   byte      `json:"toCompartmentType"`
	ToInventoryType     string    `json:"toInventoryType"`
	ReferenceId         uint32    `json:"referenceId"`
}

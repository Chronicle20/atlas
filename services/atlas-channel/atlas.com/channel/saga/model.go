package saga

import (
	"github.com/google/uuid"

	sharedsaga "github.com/Chronicle20/atlas-saga"
)

// Re-export types from atlas-saga shared library
type (
	Type   = sharedsaga.Type
	Saga   = sharedsaga.Saga
	Status = sharedsaga.Status
	Action = sharedsaga.Action
	Step   = sharedsaga.Step[any]

	// Payload types
	AwardMesosPayload            = sharedsaga.AwardMesosPayload
	AwardAssetPayload            = sharedsaga.AwardItemActionPayload
	ItemPayload                  = sharedsaga.ItemPayload
	DestroyAssetPayload          = sharedsaga.DestroyAssetPayload
	SetHPPayload                 = sharedsaga.SetHPPayload
	DeductExperiencePayload      = sharedsaga.DeductExperiencePayload
	CancelAllBuffsPayload        = sharedsaga.CancelAllBuffsPayload
	WarpToPortalPayload          = sharedsaga.WarpToPortalPayload
	ApplyConsumableEffectPayload = sharedsaga.ApplyConsumableEffectPayload
	FieldEffectWeatherPayload    = sharedsaga.FieldEffectWeatherPayload

	// Storage payload types
	DepositToStoragePayload      = sharedsaga.DepositToStoragePayload
	UpdateStorageMesosPayload    = sharedsaga.UpdateStorageMesosPayload
	TransferToStoragePayload     = sharedsaga.TransferToStoragePayload
	WithdrawFromStoragePayload   = sharedsaga.WithdrawFromStoragePayload
	WithdrawFromCashShopPayload  = sharedsaga.WithdrawFromCashShopPayload
)

// Re-export constants from atlas-saga shared library
const (
	// Saga types
	InventoryTransaction = sharedsaga.InventoryTransaction
	StorageOperation     = sharedsaga.StorageOperation
	CashShopOperation    = sharedsaga.CashShopOperation
	CharacterRespawn     = sharedsaga.CharacterRespawn
	FieldEffectUse       = sharedsaga.FieldEffectUse

	// Status constants
	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed

	// Action constants
	AwardMesos           = sharedsaga.AwardMesos
	UpdateStorageMesos   = sharedsaga.UpdateStorageMesos
	AwardAsset           = sharedsaga.AwardAsset
	DestroyAsset         = sharedsaga.DestroyAsset
	DepositToStorage     = sharedsaga.DepositToStorage
	TransferToStorage    = sharedsaga.TransferToStorage
	WithdrawFromStorage  = sharedsaga.WithdrawFromStorage
	TransferToCashShop   = sharedsaga.TransferToCashShop
	WithdrawFromCashShop = sharedsaga.WithdrawFromCashShop
	AcceptToStorage      = sharedsaga.AcceptToStorage
	ReleaseFromCharacter = sharedsaga.ReleaseFromCharacter
	AcceptToCharacter    = sharedsaga.AcceptToCharacter
	ReleaseFromStorage   = sharedsaga.ReleaseFromStorage
	SetHP                = sharedsaga.SetHP
	DeductExperience     = sharedsaga.DeductExperience
	CancelAllBuffs       = sharedsaga.CancelAllBuffs
	WarpToPortal         = sharedsaga.WarpToPortal
	FieldEffectWeather   = sharedsaga.FieldEffectWeather
	ApplyConsumableEffect = sharedsaga.ApplyConsumableEffect
)

// TransferToCashShopPayload is kept local because CashId is uint64 here
// but int64 in the shared library. This divergence needs a separate resolution.
type TransferToCashShopPayload struct {
	TransactionId       uuid.UUID `json:"transactionId"`       // Saga transaction ID
	CharacterId         uint32    `json:"characterId"`         // Character initiating the transfer
	AccountId           uint32    `json:"accountId"`           // Account ID (cash shop owner)
	CashId              uint64    `json:"cashId"`              // Cash serial number of the item to transfer
	SourceInventoryType byte      `json:"sourceInventoryType"` // Character inventory type (equip, use, etc.)
	CompartmentType     byte      `json:"compartmentType"`     // Cash shop compartment type (1=Explorer, 2=Cygnus, 3=Legend)
}

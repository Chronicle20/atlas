package saga

import (
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
	AwardItemActionPayload    = sharedsaga.AwardItemActionPayload
	ItemPayload               = sharedsaga.ItemPayload
	WarpToRandomPortalPayload = sharedsaga.WarpToRandomPortalPayload
	WarpToPortalPayload       = sharedsaga.WarpToPortalPayload
	AwardExperiencePayload    = sharedsaga.AwardExperiencePayload
	AwardLevelPayload         = sharedsaga.AwardLevelPayload
	AwardMesosPayload         = sharedsaga.AwardMesosPayload
	AwardCurrencyPayload      = sharedsaga.AwardCurrencyPayload
	DestroyAssetPayload       = sharedsaga.DestroyAssetPayload
	ChangeJobPayload          = sharedsaga.ChangeJobPayload
	CreateSkillPayload        = sharedsaga.CreateSkillPayload
	UpdateSkillPayload        = sharedsaga.UpdateSkillPayload
	ApplyConsumableEffectPayload = sharedsaga.ApplyConsumableEffectPayload
	ExperienceDistributions   = sharedsaga.ExperienceDistributions
)

// Re-export constants from atlas-saga shared library
const (
	// Saga types
	InventoryTransaction = sharedsaga.InventoryTransaction
	QuestReward          = sharedsaga.QuestReward
	TradeTransaction     = sharedsaga.TradeTransaction

	// Status constants
	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed

	// Action constants
	AwardAsset            = sharedsaga.AwardAsset
	AwardExperience       = sharedsaga.AwardExperience
	AwardLevel            = sharedsaga.AwardLevel
	AwardMesos            = sharedsaga.AwardMesos
	AwardCurrency         = sharedsaga.AwardCurrency
	WarpToRandomPortal    = sharedsaga.WarpToRandomPortal
	WarpToPortal          = sharedsaga.WarpToPortal
	DestroyAsset          = sharedsaga.DestroyAsset
	ChangeJob             = sharedsaga.ChangeJob
	CreateSkill           = sharedsaga.CreateSkill
	UpdateSkill           = sharedsaga.UpdateSkill
	ApplyConsumableEffect = sharedsaga.ApplyConsumableEffect
)

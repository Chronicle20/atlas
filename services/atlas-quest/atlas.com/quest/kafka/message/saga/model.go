package saga

import (
	sharedsaga "github.com/Chronicle20/atlas-saga"
)

// Re-export types from atlas-saga shared library
type (
	Saga                    = sharedsaga.Saga
	Step                    = sharedsaga.Step[any]
	AwardExperiencePayload  = sharedsaga.AwardExperiencePayload
	AwardMesosPayload       = sharedsaga.AwardMesosPayload
	AwardFamePayload        = sharedsaga.AwardFamePayload
	CreateSkillPayload      = sharedsaga.CreateSkillPayload
	ExperienceDistribution  = sharedsaga.ExperienceDistributions

	// Backward-compatible aliases for quest-specific naming
	AwardItemPayload   = sharedsaga.AwardItemActionPayload
	ItemDetail         = sharedsaga.ItemPayload
	ConsumeItemPayload = sharedsaga.DestroyAssetPayload
)

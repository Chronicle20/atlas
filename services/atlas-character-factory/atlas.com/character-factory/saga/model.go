package saga

import (
	"atlas-character-factory/validation"

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
	AwardItemActionPayload                = sharedsaga.AwardItemActionPayload
	ItemPayload                           = sharedsaga.ItemPayload
	WarpToRandomPortalPayload             = sharedsaga.WarpToRandomPortalPayload
	WarpToPortalPayload                   = sharedsaga.WarpToPortalPayload
	AwardExperiencePayload                = sharedsaga.AwardExperiencePayload
	AwardLevelPayload                     = sharedsaga.AwardLevelPayload
	AwardMesosPayload                     = sharedsaga.AwardMesosPayload
	DestroyAssetPayload                   = sharedsaga.DestroyAssetPayload
	EquipAssetPayload                     = sharedsaga.EquipAssetPayload
	UnequipAssetPayload                   = sharedsaga.UnequipAssetPayload
	ChangeJobPayload                      = sharedsaga.ChangeJobPayload
	CreateSkillPayload                    = sharedsaga.CreateSkillPayload
	UpdateSkillPayload                    = sharedsaga.UpdateSkillPayload
	RequestGuildNamePayload               = sharedsaga.RequestGuildNamePayload
	RequestGuildEmblemPayload             = sharedsaga.RequestGuildEmblemPayload
	RequestGuildDisbandPayload            = sharedsaga.RequestGuildDisbandPayload
	RequestGuildCapacityIncreasePayload   = sharedsaga.RequestGuildCapacityIncreasePayload
	CreateInvitePayload                   = sharedsaga.CreateInvitePayload
	CharacterCreatePayload                = sharedsaga.CharacterCreatePayload
	CreateAndEquipAssetPayload            = sharedsaga.CreateAndEquipAssetPayload
	AwaitCharacterCreatedPayload          = sharedsaga.AwaitCharacterCreatedPayload
	ExperienceDistributions               = sharedsaga.ExperienceDistributions
)

// Re-export constants from atlas-saga shared library
const (
	// Saga types
	InventoryTransaction      = sharedsaga.InventoryTransaction
	QuestReward               = sharedsaga.QuestReward
	TradeTransaction          = sharedsaga.TradeTransaction
	CharacterCreation         = sharedsaga.CharacterCreation
	CharacterCreationOnly     = sharedsaga.CharacterCreationOnly
	CharacterCreationFollowUp = sharedsaga.CharacterCreationFollowUp

	// Status constants
	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed

	// Action constants
	AwardAsset                   = sharedsaga.AwardAsset
	AwardExperience              = sharedsaga.AwardExperience
	AwardLevel                   = sharedsaga.AwardLevel
	AwardMesos                   = sharedsaga.AwardMesos
	WarpToRandomPortal           = sharedsaga.WarpToRandomPortal
	WarpToPortal                 = sharedsaga.WarpToPortal
	DestroyAsset                 = sharedsaga.DestroyAsset
	EquipAsset                   = sharedsaga.EquipAsset
	UnequipAsset                 = sharedsaga.UnequipAsset
	ChangeJob                    = sharedsaga.ChangeJob
	CreateSkill                  = sharedsaga.CreateSkill
	UpdateSkill                  = sharedsaga.UpdateSkill
	ValidateCharacterState       = sharedsaga.ValidateCharacterState
	RequestGuildName             = sharedsaga.RequestGuildName
	RequestGuildEmblem           = sharedsaga.RequestGuildEmblem
	RequestGuildDisband          = sharedsaga.RequestGuildDisband
	RequestGuildCapacityIncrease = sharedsaga.RequestGuildCapacityIncrease
	CreateInvite                 = sharedsaga.CreateInvite
	CreateCharacter              = sharedsaga.CreateCharacter
	CreateAndEquipAsset          = sharedsaga.CreateAndEquipAsset
	AwaitCharacterCreated        = sharedsaga.AwaitCharacterCreated
)

// ValidateCharacterStatePayload uses the character-factory service's validation.ConditionInput.
// This is service-specific and wraps the shared ValidationConditionInput with the local type.
type ValidateCharacterStatePayload struct {
	CharacterId uint32                      `json:"characterId"`
	Conditions  []validation.ConditionInput `json:"conditions"`
}

// ToSharedPayload converts to the shared saga payload type
func (p ValidateCharacterStatePayload) ToSharedPayload() sharedsaga.ValidateCharacterStatePayload {
	conditions := make([]sharedsaga.ValidationConditionInput, len(p.Conditions))
	for i, c := range p.Conditions {
		conditions[i] = sharedsaga.ValidationConditionInput{
			Type:        c.Type,
			Operator:    c.Operator,
			Value:       c.Value,
			ReferenceId: c.ItemId,
		}
	}
	return sharedsaga.ValidateCharacterStatePayload{
		CharacterId: p.CharacterId,
		Conditions:  conditions,
	}
}

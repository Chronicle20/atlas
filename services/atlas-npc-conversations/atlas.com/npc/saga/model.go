package saga

import (
	"atlas-npc-conversations/validation"

	scriptsaga "github.com/Chronicle20/atlas-script-core/saga"
)

// Re-export types from atlas-script-core/saga
type (
	Type   = scriptsaga.Type
	Saga   = scriptsaga.Saga
	Status = scriptsaga.Status
	Action = scriptsaga.Action

	// Payload types
	AwardItemActionPayload       = scriptsaga.AwardItemActionPayload
	ItemPayload                  = scriptsaga.ItemPayload
	WarpToRandomPortalPayload    = scriptsaga.WarpToRandomPortalPayload
	WarpToPortalPayload          = scriptsaga.WarpToPortalPayload
	AwardExperiencePayload       = scriptsaga.AwardExperiencePayload
	AwardLevelPayload            = scriptsaga.AwardLevelPayload
	AwardMesosPayload            = scriptsaga.AwardMesosPayload
	DestroyAssetPayload          = scriptsaga.DestroyAssetPayload
	ChangeJobPayload             = scriptsaga.ChangeJobPayload
	CreateSkillPayload           = scriptsaga.CreateSkillPayload
	UpdateSkillPayload           = scriptsaga.UpdateSkillPayload
	IncreaseBuddyCapacityPayload = scriptsaga.IncreaseBuddyCapacityPayload
	GainClosenessPayload         = scriptsaga.GainClosenessPayload
	ChangeHairPayload            = scriptsaga.ChangeHairPayload
	ChangeFacePayload            = scriptsaga.ChangeFacePayload
	ChangeSkinPayload            = scriptsaga.ChangeSkinPayload
	SpawnMonsterPayload          = scriptsaga.SpawnMonsterPayload
	CompleteQuestPayload         = scriptsaga.CompleteQuestPayload
	StartQuestPayload            = scriptsaga.StartQuestPayload
	ApplyConsumableEffectPayload = scriptsaga.ApplyConsumableEffectPayload
	SendMessagePayload           = scriptsaga.SendMessagePayload
	AwardFamePayload             = scriptsaga.AwardFamePayload
	ShowStoragePayload           = scriptsaga.ShowStoragePayload
	ExperienceDistributions      = scriptsaga.ExperienceDistributions

	// Portal-specific payload types (re-exported for completeness)
	PlayPortalSoundPayload  = scriptsaga.PlayPortalSoundPayload
	UpdateAreaInfoPayload   = scriptsaga.UpdateAreaInfoPayload
	ShowInfoPayload         = scriptsaga.ShowInfoPayload
	ShowInfoTextPayload     = scriptsaga.ShowInfoTextPayload
	ShowHintPayload         = scriptsaga.ShowHintPayload
)

// Re-export constants from atlas-script-core/saga
const (
	// Saga types
	InventoryTransaction = scriptsaga.InventoryTransaction
	QuestReward          = scriptsaga.QuestReward
	TradeTransaction     = scriptsaga.TradeTransaction

	// Status constants
	Pending   = scriptsaga.Pending
	Completed = scriptsaga.Completed
	Failed    = scriptsaga.Failed

	// Action constants
	AwardInventory         = scriptsaga.AwardInventory
	AwardExperience        = scriptsaga.AwardExperience
	AwardLevel             = scriptsaga.AwardLevel
	AwardMesos             = scriptsaga.AwardMesos
	WarpToRandomPortal     = scriptsaga.WarpToRandomPortal
	WarpToPortal           = scriptsaga.WarpToPortal
	DestroyAsset           = scriptsaga.DestroyAsset
	ChangeJob              = scriptsaga.ChangeJob
	CreateSkill            = scriptsaga.CreateSkill
	UpdateSkill            = scriptsaga.UpdateSkill
	ValidateCharacterState = scriptsaga.ValidateCharacterState
	IncreaseBuddyCapacity  = scriptsaga.IncreaseBuddyCapacity
	GainCloseness          = scriptsaga.GainCloseness
	ChangeHair             = scriptsaga.ChangeHair
	ChangeFace             = scriptsaga.ChangeFace
	ChangeSkin             = scriptsaga.ChangeSkin
	SpawnMonster           = scriptsaga.SpawnMonster
	CompleteQuest          = scriptsaga.CompleteQuest
	StartQuest             = scriptsaga.StartQuest
	ApplyConsumableEffect  = scriptsaga.ApplyConsumableEffect
	SendMessage            = scriptsaga.SendMessage
	AwardFame              = scriptsaga.AwardFame
	ShowStorage            = scriptsaga.ShowStorage

	// Portal-specific actions (re-exported for completeness)
	PlayPortalSound = scriptsaga.PlayPortalSound
	UpdateAreaInfo  = scriptsaga.UpdateAreaInfo
	ShowInfo        = scriptsaga.ShowInfo
	ShowInfoText    = scriptsaga.ShowInfoText
	ShowHint        = scriptsaga.ShowHint
)

// ValidateCharacterStatePayload uses the NPC service's validation.ConditionInput
// This is NPC-specific and maps to the shared ValidationConditionInput
type ValidateCharacterStatePayload struct {
	CharacterId uint32                      `json:"characterId"`
	Conditions  []validation.ConditionInput `json:"conditions"`
}

// ToSharedPayload converts to the shared saga payload type
func (p ValidateCharacterStatePayload) ToSharedPayload() scriptsaga.ValidateCharacterStatePayload {
	conditions := make([]scriptsaga.ValidationConditionInput, len(p.Conditions))
	for i, c := range p.Conditions {
		conditions[i] = scriptsaga.ValidationConditionInput{
			Type:            c.Type,
			Operator:        c.Operator,
			Value:           c.Value,
			ReferenceId:     c.ReferenceId,
			Step:            c.Step,
			WorldId:         c.WorldId,
			ChannelId:       c.ChannelId,
			IncludeEquipped: c.IncludeEquipped,
		}
	}
	return scriptsaga.ValidateCharacterStatePayload{
		CharacterId: p.CharacterId,
		Conditions:  conditions,
	}
}

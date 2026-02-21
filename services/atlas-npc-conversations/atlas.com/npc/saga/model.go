package saga

import (
	"atlas-npc-conversations/validation"

	sharedsaga "github.com/Chronicle20/atlas-saga"
)

// Re-export types from atlas-saga shared library
type (
	Type   = sharedsaga.Type
	Saga   = sharedsaga.Saga
	Status = sharedsaga.Status
	Action = sharedsaga.Action

	// Payload types
	AwardItemActionPayload       = sharedsaga.AwardItemActionPayload
	ItemPayload                  = sharedsaga.ItemPayload
	WarpToRandomPortalPayload    = sharedsaga.WarpToRandomPortalPayload
	WarpToPortalPayload          = sharedsaga.WarpToPortalPayload
	AwardExperiencePayload       = sharedsaga.AwardExperiencePayload
	AwardLevelPayload            = sharedsaga.AwardLevelPayload
	AwardMesosPayload            = sharedsaga.AwardMesosPayload
	DestroyAssetPayload          = sharedsaga.DestroyAssetPayload
	DestroyAssetFromSlotPayload  = sharedsaga.DestroyAssetFromSlotPayload
	ChangeJobPayload             = sharedsaga.ChangeJobPayload
	CreateSkillPayload           = sharedsaga.CreateSkillPayload
	UpdateSkillPayload           = sharedsaga.UpdateSkillPayload
	IncreaseBuddyCapacityPayload = sharedsaga.IncreaseBuddyCapacityPayload
	GainClosenessPayload         = sharedsaga.GainClosenessPayload
	ChangeHairPayload            = sharedsaga.ChangeHairPayload
	ChangeFacePayload            = sharedsaga.ChangeFacePayload
	ChangeSkinPayload            = sharedsaga.ChangeSkinPayload
	SpawnMonsterPayload          = sharedsaga.SpawnMonsterPayload
	CompleteQuestPayload         = sharedsaga.CompleteQuestPayload
	StartQuestPayload            = sharedsaga.StartQuestPayload
	SetQuestProgressPayload      = sharedsaga.SetQuestProgressPayload
	ApplyConsumableEffectPayload = sharedsaga.ApplyConsumableEffectPayload
	SendMessagePayload           = sharedsaga.SendMessagePayload
	AwardFamePayload             = sharedsaga.AwardFamePayload
	ShowStoragePayload           = sharedsaga.ShowStoragePayload
	ExperienceDistributions      = sharedsaga.ExperienceDistributions

	// Portal-specific payload types
	PlayPortalSoundPayload = sharedsaga.PlayPortalSoundPayload
	UpdateAreaInfoPayload  = sharedsaga.UpdateAreaInfoPayload
	ShowInfoPayload        = sharedsaga.ShowInfoPayload
	ShowInfoTextPayload    = sharedsaga.ShowInfoTextPayload
	ShowHintPayload        = sharedsaga.ShowHintPayload
	ShowGuideHintPayload   = sharedsaga.ShowGuideHintPayload
	ShowIntroPayload       = sharedsaga.ShowIntroPayload
	SetHPPayload           = sharedsaga.SetHPPayload
	ResetStatsPayload      = sharedsaga.ResetStatsPayload

	// Saved location payload types
	SaveLocationPayload        = sharedsaga.SaveLocationPayload
	WarpToSavedLocationPayload = sharedsaga.WarpToSavedLocationPayload

	// Gachapon payload types
	SelectGachaponRewardPayload = sharedsaga.SelectGachaponRewardPayload

	// Party quest payload types
	RegisterPartyQuestPayload         = sharedsaga.RegisterPartyQuestPayload
	WarpPartyQuestMembersToMapPayload = sharedsaga.WarpPartyQuestMembersToMapPayload
	LeavePartyQuestPayload            = sharedsaga.LeavePartyQuestPayload
	StageClearAttemptPqPayload        = sharedsaga.StageClearAttemptPqPayload
	EnterPartyQuestBonusPayload       = sharedsaga.EnterPartyQuestBonusPayload

	// Transport payload types
	StartInstanceTransportPayload = sharedsaga.StartInstanceTransportPayload
)

// Re-export constants from atlas-saga shared library
const (
	// Saga types
	InventoryTransaction = sharedsaga.InventoryTransaction
	QuestReward          = sharedsaga.QuestReward
	TradeTransaction     = sharedsaga.TradeTransaction
	GachaponTransaction  = sharedsaga.GachaponTransaction

	// Status constants
	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed

	// Action constants
	AwardAsset             = sharedsaga.AwardAsset
	AwardExperience        = sharedsaga.AwardExperience
	AwardLevel             = sharedsaga.AwardLevel
	AwardMesos             = sharedsaga.AwardMesos
	WarpToRandomPortal     = sharedsaga.WarpToRandomPortal
	WarpToPortal           = sharedsaga.WarpToPortal
	DestroyAsset           = sharedsaga.DestroyAsset
	DestroyAssetFromSlot   = sharedsaga.DestroyAssetFromSlot
	ChangeJob              = sharedsaga.ChangeJob
	CreateSkill            = sharedsaga.CreateSkill
	UpdateSkill            = sharedsaga.UpdateSkill
	ValidateCharacterState = sharedsaga.ValidateCharacterState
	IncreaseBuddyCapacity  = sharedsaga.IncreaseBuddyCapacity
	GainCloseness          = sharedsaga.GainCloseness
	ChangeHair             = sharedsaga.ChangeHair
	ChangeFace             = sharedsaga.ChangeFace
	ChangeSkin             = sharedsaga.ChangeSkin
	SpawnMonster           = sharedsaga.SpawnMonster
	CompleteQuest          = sharedsaga.CompleteQuest
	StartQuest             = sharedsaga.StartQuest
	SetQuestProgress       = sharedsaga.SetQuestProgress
	ApplyConsumableEffect  = sharedsaga.ApplyConsumableEffect
	SendMessage            = sharedsaga.SendMessage
	AwardFame              = sharedsaga.AwardFame
	ShowStorage            = sharedsaga.ShowStorage

	// Portal-specific actions
	PlayPortalSound = sharedsaga.PlayPortalSound
	UpdateAreaInfo  = sharedsaga.UpdateAreaInfo
	ShowInfo        = sharedsaga.ShowInfo
	ShowInfoText    = sharedsaga.ShowInfoText
	ShowHint        = sharedsaga.ShowHint

	// Character stat actions
	ShowGuideHint = sharedsaga.ShowGuideHint
	ShowIntro     = sharedsaga.ShowIntro
	SetHP         = sharedsaga.SetHP
	ResetStats    = sharedsaga.ResetStats

	// Transport actions
	StartInstanceTransport = sharedsaga.StartInstanceTransport

	// Party quest actions
	RegisterPartyQuest         = sharedsaga.RegisterPartyQuest
	WarpPartyQuestMembersToMap = sharedsaga.WarpPartyQuestMembersToMap
	LeavePartyQuest            = sharedsaga.LeavePartyQuest
	StageClearAttemptPq        = sharedsaga.StageClearAttemptPq
	EnterPartyQuestBonus       = sharedsaga.EnterPartyQuestBonus

	// Gachapon actions
	SelectGachaponReward = sharedsaga.SelectGachaponReward

	// Saved location actions
	SaveLocation        = sharedsaga.SaveLocation
	WarpToSavedLocation = sharedsaga.WarpToSavedLocation
)

// ValidateCharacterStatePayload uses the NPC service's validation.ConditionInput.
// This is NPC-specific and wraps the shared ValidationConditionInput with the local type.
type ValidateCharacterStatePayload struct {
	CharacterId uint32                      `json:"characterId"`
	Conditions  []validation.ConditionInput `json:"conditions"`
}

// ToSharedPayload converts to the shared saga payload type
func (p ValidateCharacterStatePayload) ToSharedPayload() sharedsaga.ValidateCharacterStatePayload {
	conditions := make([]sharedsaga.ValidationConditionInput, len(p.Conditions))
	for i, c := range p.Conditions {
		conditions[i] = sharedsaga.ValidationConditionInput{
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
	return sharedsaga.ValidateCharacterStatePayload{
		CharacterId: p.CharacterId,
		Conditions:  conditions,
	}
}

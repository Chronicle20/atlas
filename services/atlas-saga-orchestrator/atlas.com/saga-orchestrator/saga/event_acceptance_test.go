package saga

import (
	"testing"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// allActions lists every Action constant from libs/atlas-saga/model.go (lines
// 43–157). Keep this list in sync when new Actions are added. The coverage
// test below fails if acceptanceTable lacks an entry for any of these.
var allActions = []sharedsaga.Action{
	sharedsaga.AwardAsset, sharedsaga.AwardExperience, sharedsaga.AwardLevel, sharedsaga.AwardMesos,
	sharedsaga.AwardCurrency, sharedsaga.AwardFame, sharedsaga.DestroyAsset, sharedsaga.DestroyAssetFromSlot,
	sharedsaga.EquipAsset, sharedsaga.UnequipAsset, sharedsaga.CreateAndEquipAsset,
	sharedsaga.WarpToRandomPortal, sharedsaga.WarpToPortal, sharedsaga.WarpToSavedLocation, sharedsaga.SaveLocation,
	sharedsaga.ChangeJob, sharedsaga.ChangeHair, sharedsaga.ChangeFace, sharedsaga.ChangeSkin, sharedsaga.SetHP,
	sharedsaga.DeductExperience, sharedsaga.CancelAllBuffs, sharedsaga.ResetStats, sharedsaga.RebalanceAP,
	sharedsaga.ValidateCharacterState, sharedsaga.IncreaseBuddyCapacity, sharedsaga.GainCloseness,
	sharedsaga.CreateSkill, sharedsaga.UpdateSkill,
	sharedsaga.CompleteQuest, sharedsaga.StartQuest, sharedsaga.SetQuestProgress, sharedsaga.ForfeitQuest,
	sharedsaga.ApplyConsumableEffect, sharedsaga.CancelConsumableEffect,
	sharedsaga.SendMessage, sharedsaga.FieldEffect, sharedsaga.UiLock, sharedsaga.PlayPortalSound,
	sharedsaga.UpdateAreaInfo, sharedsaga.ShowInfo, sharedsaga.ShowInfoText, sharedsaga.ShowIntro,
	sharedsaga.ShowHint, sharedsaga.ShowGuideHint, sharedsaga.BlockPortal, sharedsaga.UnblockPortal,
	sharedsaga.SpawnMonster, sharedsaga.SpawnReactorDrops,
	sharedsaga.ShowStorage, sharedsaga.DepositToStorage, sharedsaga.UpdateStorageMesos,
	sharedsaga.TransferToStorage, sharedsaga.WithdrawFromStorage, sharedsaga.AcceptToStorage,
	sharedsaga.ReleaseFromCharacter, sharedsaga.AcceptToCharacter, sharedsaga.ReleaseFromStorage,
	sharedsaga.TransferToCashShop, sharedsaga.WithdrawFromCashShop, sharedsaga.AcceptToCashShop,
	sharedsaga.ReleaseFromCashShop,
	sharedsaga.RequestGuildName, sharedsaga.RequestGuildEmblem, sharedsaga.RequestGuildDisband,
	sharedsaga.RequestGuildCapacityIncrease, sharedsaga.CreateInvite,
	sharedsaga.CreateCharacter, sharedsaga.AwaitCharacterCreated,
	sharedsaga.StartInstanceTransport,
	sharedsaga.SelectGachaponReward, sharedsaga.EmitGachaponWin,
	sharedsaga.RegisterPartyQuest, sharedsaga.WarpPartyQuestMembersToMap, sharedsaga.LeavePartyQuest,
	sharedsaga.EnterPartyQuestBonus, sharedsaga.UpdatePqCustomData, sharedsaga.HitReactor,
	sharedsaga.BroadcastPqMessage, sharedsaga.StageClearAttemptPq, sharedsaga.FieldEffectWeather,
}

// TestAcceptanceTable_EveryActionRepresented asserts every Action constant
// declared in libs/atlas-saga/model.go has an acceptanceTable entry
// (possibly empty, for self-completing or composite actions).
func TestAcceptanceTable_EveryActionRepresented(t *testing.T) {
	for _, a := range allActions {
		if _, ok := acceptanceTable[a]; !ok {
			t.Errorf("acceptanceTable missing entry for Action %q — add it to event_acceptance.go (or use an empty slice for self-completing actions)", a)
		}
	}
}

// TestStepAcceptsEvent_KnownSuccessKinds locks in the 1:1 success mappings
// the design §2.2 / PRD §4.1 enumerate. If any of these flips, a real saga
// flow is broken.
func TestStepAcceptsEvent_KnownSuccessKinds(t *testing.T) {
	cases := []struct {
		action sharedsaga.Action
		kind   EventKind
	}{
		{sharedsaga.AwardExperience, EventKindCharacterExperienceChanged},
		{sharedsaga.AwardLevel, EventKindCharacterLevelChanged},
		{sharedsaga.AwardMesos, EventKindCharacterMesoChanged},
		{sharedsaga.ChangeHair, EventKindCharacterStatChanged},
		{sharedsaga.ChangeFace, EventKindCharacterStatChanged},
		{sharedsaga.ChangeSkin, EventKindCharacterStatChanged},
		{sharedsaga.SetHP, EventKindCharacterStatChanged},
		{sharedsaga.ResetStats, EventKindCharacterStatChanged},
		{sharedsaga.DeductExperience, EventKindCharacterExperienceChanged},
		{sharedsaga.CreateCharacter, EventKindCharacterCreated},
		{sharedsaga.CreateAndEquipAsset, EventKindAssetCreated},
		{sharedsaga.DestroyAsset, EventKindAssetDeleted},
		{sharedsaga.DestroyAssetFromSlot, EventKindAssetDeleted},
		{sharedsaga.EquipAsset, EventKindAssetMoved},
		{sharedsaga.UnequipAsset, EventKindAssetMoved},
		{sharedsaga.CreateSkill, EventKindSkillCreated},
		{sharedsaga.UpdateSkill, EventKindSkillUpdated},
		{sharedsaga.CompleteQuest, EventKindQuestCompleted},
		{sharedsaga.StartQuest, EventKindQuestStarted},
		{sharedsaga.ForfeitQuest, EventKindQuestForfeited},
		{sharedsaga.ApplyConsumableEffect, EventKindConsumableEffectApplied},
		{sharedsaga.IncreaseBuddyCapacity, EventKindBuddyCapacityChanged},
		{sharedsaga.GainCloseness, EventKindPetClosenessChanged},
		{sharedsaga.UpdateStorageMesos, EventKindStorageMesosUpdated},
		{sharedsaga.AcceptToStorage, EventKindStorageCompartmentAccepted},
		{sharedsaga.ReleaseFromStorage, EventKindStorageCompartmentReleased},
		{sharedsaga.AcceptToCharacter, EventKindCompartmentAccepted},
		{sharedsaga.ReleaseFromCharacter, EventKindCompartmentReleased},
		{sharedsaga.AcceptToCashShop, EventKindCashShopCompartmentAccepted},
		{sharedsaga.ReleaseFromCashShop, EventKindCashShopCompartmentReleased},
		{sharedsaga.RequestGuildName, EventKindGuildRequestAgreement},
		{sharedsaga.RequestGuildEmblem, EventKindGuildEmblemUpdated},
		{sharedsaga.RequestGuildDisband, EventKindGuildDisbanded},
		{sharedsaga.RequestGuildCapacityIncrease, EventKindGuildCapacityUpdated},
		{sharedsaga.CreateInvite, EventKindInviteCreated},
	}
	for _, tc := range cases {
		if !StepAcceptsEvent(tc.action, tc.kind) {
			t.Errorf("StepAcceptsEvent(%q, %q) = false; want true", tc.action, tc.kind)
		}
	}
}

// TestStepAcceptsEvent_FailureKinds ensures failure events gate on the
// actions that legitimately fail through them.
func TestStepAcceptsEvent_FailureKinds(t *testing.T) {
	cases := []struct {
		action sharedsaga.Action
		kind   EventKind
	}{
		{sharedsaga.CreateCharacter, EventKindCharacterCreationFailed},
		{sharedsaga.AwardMesos, EventKindCharacterMesoError},
		{sharedsaga.AwardAsset, EventKindAssetQuantityChanged},
		{sharedsaga.DestroyAsset, EventKindAssetQuantityChanged},
		{sharedsaga.DestroyAssetFromSlot, EventKindAssetQuantityChanged},
	}
	for _, tc := range cases {
		if !StepAcceptsEvent(tc.action, tc.kind) {
			t.Errorf("StepAcceptsEvent(%q, %q) = false; want true", tc.action, tc.kind)
		}
	}
}

func TestStepAcceptsEvent_BugClassAntiMatches(t *testing.T) {
	// The STAT_CHANGED ripple in §9.1 must not complete AwardAsset,
	// CreateAndEquipAsset, or ChangeJob steps.
	cases := []struct {
		action sharedsaga.Action
		kind   EventKind
	}{
		{sharedsaga.AwardAsset, EventKindCharacterStatChanged},
		{sharedsaga.CreateAndEquipAsset, EventKindCharacterStatChanged},
		{sharedsaga.ChangeJob, EventKindCharacterStatChanged},
	}
	for _, tc := range cases {
		if StepAcceptsEvent(tc.action, tc.kind) {
			t.Errorf("StepAcceptsEvent(%q, %q) = true; want false (this is the §9.1 bug)", tc.action, tc.kind)
		}
	}
}

func TestStepAcceptsEvent_ThiefScenarioMatches(t *testing.T) {
	cases := []struct {
		action sharedsaga.Action
		kind   EventKind
	}{
		{sharedsaga.RebalanceAP, EventKindCharacterStatChanged},
		{sharedsaga.ChangeJob, EventKindCharacterJobChanged},
		{sharedsaga.AwardAsset, EventKindAssetCreated},
	}
	for _, tc := range cases {
		if !StepAcceptsEvent(tc.action, tc.kind) {
			t.Errorf("StepAcceptsEvent(%q, %q) = false; want true", tc.action, tc.kind)
		}
	}
}

func TestStepAcceptsEvent_UnknownActionDefaultDenies(t *testing.T) {
	if StepAcceptsEvent(sharedsaga.Action("nonexistent_action"), EventKindAssetCreated) {
		t.Errorf("StepAcceptsEvent should default-deny unknown actions")
	}
}

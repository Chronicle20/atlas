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

// TestAcceptanceTable_EveryActionRepresented is the Task 1 skeleton test.
// It asserts AwardAsset, ChangeJob, RebalanceAP — the three actions from the
// Thief scenario (§9.1). Task 2 extends this to the full `allActions` list.
func TestAcceptanceTable_EveryActionRepresented_Thief(t *testing.T) {
	for _, a := range []sharedsaga.Action{sharedsaga.RebalanceAP, sharedsaga.ChangeJob, sharedsaga.AwardAsset} {
		if _, ok := acceptanceTable[a]; !ok {
			t.Errorf("acceptanceTable missing entry for Action %q", a)
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

package saga

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// TestWorldTransferPayloadsRoundTrip asserts each world-transfer payload
// round-trips through JSON with the exact wire tags the orchestrator will
// unmarshal by action (task-227 task-13).
func TestWorldTransferPayloadsRoundTrip(t *testing.T) {
	for _, p := range []any{
		ValidateWorldTransferPayload{CharacterId: 1, SourceWorldId: 0, DestinationWorldId: 1, PendingChangeId: uuid.New()},
		LeaveGuildForTransferPayload{CharacterId: 1, WorldId: 0, GuildId: 5, Title: 3},
		LeavePartyForTransferPayload{CharacterId: 1, WorldId: 0, PartyId: 9},
		SeverBuddiesForTransferPayload{CharacterId: 1, WorldId: 0, BuddyIds: []uint32{2, 3}},
		ChangeCharacterWorldPayload{CharacterId: 1, SourceWorldId: 0, DestinationWorldId: 1, PendingChangeId: uuid.New()},
	} {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal %T: %v", p, err)
		}
		out := reflect.New(reflect.TypeOf(p)).Interface()
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("unmarshal %T: %v", p, err)
		}
		if !reflect.DeepEqual(p, reflect.ValueOf(out).Elem().Interface()) {
			t.Fatalf("%T did not round-trip", p)
		}
	}
}

// TestWorldTransferPayloadWireTags pins the exact JSON wire tags for each
// world-transfer payload so a renamed field is caught here rather than
// silently zero-valuing on the orchestrator side (task-13 unmarshals these
// by action).
func TestWorldTransferPayloadWireTags(t *testing.T) {
	cases := []struct {
		name string
		v    any
		want string
	}{
		{
			"ValidateWorldTransferPayload",
			ValidateWorldTransferPayload{CharacterId: 1, SourceWorldId: 2, DestinationWorldId: 3, PendingChangeId: uuid.Nil},
			`{"characterId":1,"sourceWorldId":2,"destinationWorldId":3,"pendingChangeId":"00000000-0000-0000-0000-000000000000"}`,
		},
		{
			"LeaveGuildForTransferPayload",
			LeaveGuildForTransferPayload{CharacterId: 1, WorldId: 2, GuildId: 3, Title: 4},
			`{"characterId":1,"worldId":2,"guildId":3,"title":4}`,
		},
		{
			"LeavePartyForTransferPayload",
			LeavePartyForTransferPayload{CharacterId: 1, WorldId: 2, PartyId: 3},
			`{"characterId":1,"worldId":2,"partyId":3}`,
		},
		{
			"SeverBuddiesForTransferPayload",
			SeverBuddiesForTransferPayload{CharacterId: 1, WorldId: 2, BuddyIds: []uint32{3, 4}},
			`{"characterId":1,"worldId":2,"buddyIds":[3,4]}`,
		},
		{
			"ChangeCharacterWorldPayload",
			ChangeCharacterWorldPayload{CharacterId: 1, SourceWorldId: 2, DestinationWorldId: 3, PendingChangeId: uuid.Nil},
			`{"characterId":1,"sourceWorldId":2,"destinationWorldId":3,"pendingChangeId":"00000000-0000-0000-0000-000000000000"}`,
		},
	}

	for _, c := range cases {
		b, err := json.Marshal(c.v)
		if err != nil {
			t.Fatalf("%s: marshal: %v", c.name, err)
		}
		if string(b) != c.want {
			t.Fatalf("%s: wire mismatch\n got: %s\nwant: %s", c.name, string(b), c.want)
		}
	}
}

// TestWorldTransferActionConstantsAreUnique asserts none of the five new
// world-transfer Action constants collide with each other or with any
// pre-existing Action constant in this package.
func TestWorldTransferActionConstantsAreUnique(t *testing.T) {
	worldTransferActions := []Action{
		ValidateWorldTransfer,
		LeaveGuildForTransfer,
		LeavePartyForTransfer,
		SeverBuddiesForTransfer,
		ChangeCharacterWorld,
	}

	otherActions := []Action{
		AwardAsset, AwardExperience, AwardLevel, AwardMesos, AwardCurrency, AwardFame,
		DestroyAsset, DestroyAssetFromSlot, EquipAsset, UnequipAsset, CreateAndEquipAsset,
		WarpToRandomPortal, WarpToPortal, WarpToSavedLocation, SaveLocation,
		ChangeJob, ChangeHair, ChangeFace, ChangeSkin, SetHP, DeductExperience,
		CancelAllBuffs, ResetStats, RebalanceAP, ValidateCharacterState,
		IncreaseBuddyCapacity, GainCloseness, EvolvePet, TransferAP, TransferSP,
		CreateSkill, UpdateSkill,
		CompleteQuest, StartQuest, SetQuestProgress, ForfeitQuest,
		ApplyConsumableEffect, CancelConsumableEffect,
		SendMessage,
		FieldEffect, UiLock, PlayPortalSound, UpdateAreaInfo, ShowInfo, ShowInfoText,
		ShowIntro, ShowHint, ShowGuideHint, BlockPortal, UnblockPortal,
		SpawnMonster, SpawnReactorDrops,
		ShowStorage, DepositToStorage, UpdateStorageMesos, TransferToStorage,
		WithdrawFromStorage, AcceptToStorage, ReleaseFromCharacter, AcceptToCharacter,
		ReleaseFromStorage,
		OpenNpcShop,
		TradeSettlement, TradeUnwind, TransferToTrade, AcceptToTrade, ReleaseFromTrade,
		TransferToCashShop, WithdrawFromCashShop, AcceptToCashShop, ReleaseFromCashShop,
		TransferToMts, WithdrawFromMts, AcceptToMtsListing, ReleaseFromMtsHolding,
		MtsSettlePurchase, MtsMoveListingToHolding, MtsBidEscrow,
		RequestGuildName, RequestGuildEmblem, RequestGuildDisband,
		RequestGuildCapacityIncrease, CreateInvite,
		CreateCharacter, AwaitCharacterCreated, AwaitInventoryCreated,
		StartInstanceTransport,
		SelectGachaponReward, EmitGachaponWin,
		StartRPSGame,
		RegisterPartyQuest, WarpPartyQuestMembersToMap, LeavePartyQuest, EnterPartyQuestBonus,
		UpdatePqCustomData, HitReactor, BroadcastPqMessage, StageClearAttemptPq,
		FieldEffectWeather,
		PlayJukebox,
		CreateNote,
		SetAssetOwner, ApplyAssetLock, ExtendAssetExpiration, IncubatorResult,
		EmitMegaphone, EnqueueWorldBroadcast,
	}

	seen := make(map[Action]bool, len(worldTransferActions))
	for _, a := range worldTransferActions {
		if seen[a] {
			t.Fatalf("duplicate world-transfer action constant: %s", a)
		}
		seen[a] = true
	}

	for _, other := range otherActions {
		if seen[other] {
			t.Fatalf("world-transfer action collides with existing action constant: %s", other)
		}
	}
}

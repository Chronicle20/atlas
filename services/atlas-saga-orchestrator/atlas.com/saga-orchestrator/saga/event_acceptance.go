package saga

import (
	"github.com/sirupsen/logrus"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// EventKind is a compact tag for the semantic class of an event received on a
// status topic. Each consumer handler hardcodes one EventKind constant per
// handler function; the handler's identity is the classification.
type EventKind string

const (
	// Character subsystem.
	EventKindCharacterMapChanged        EventKind = "character.map_changed"
	EventKindCharacterExperienceChanged EventKind = "character.experience_changed"
	EventKindCharacterLevelChanged      EventKind = "character.level_changed"
	EventKindCharacterMesoChanged       EventKind = "character.meso_changed"
	EventKindCharacterJobChanged        EventKind = "character.job_changed"
	EventKindCharacterCreated           EventKind = "character.created"
	EventKindCharacterCreationFailed    EventKind = "character.creation_failed"
	EventKindCharacterStatChanged       EventKind = "character.stat_changed"
	EventKindCharacterMesoError         EventKind = "character.meso_error"
	EventKindCharacterApTransferError   EventKind = "character.ap_transfer_error"
	EventKindCharacterDeleted           EventKind = "character.deleted"

	// Asset subsystem.
	EventKindAssetCreated         EventKind = "asset.created"
	EventKindAssetDeleted         EventKind = "asset.deleted"
	EventKindAssetQuantityChanged EventKind = "asset.quantity_changed"
	EventKindAssetMoved           EventKind = "asset.moved"

	// Quest subsystem.
	EventKindQuestStarted   EventKind = "quest.started"
	EventKindQuestCompleted EventKind = "quest.completed"
	EventKindQuestForfeited EventKind = "quest.forfeited"

	// Skill subsystem.
	EventKindSkillCreated         EventKind = "skill.created"
	EventKindSkillUpdated         EventKind = "skill.updated"
	EventKindSkillDeleted         EventKind = "skill.deleted"
	EventKindSkillSpTransferred   EventKind = "skill.sp_transferred"
	EventKindSkillSpTransferError EventKind = "skill.sp_transfer_error"

	// Buddy list.
	EventKindBuddyCapacityChanged EventKind = "buddy.capacity_changed"

	// Consumable.
	EventKindConsumableEffectApplied EventKind = "consumable.effect_applied"

	// Pet.
	EventKindPetClosenessChanged EventKind = "pet.closeness_changed"
	EventKindPetEvolved          EventKind = "pet.evolved"

	// Cash shop.
	EventKindCashShopWalletUpdated       EventKind = "cashshop.wallet_updated"
	EventKindCashShopWalletError         EventKind = "cashshop.wallet_error"
	EventKindCashShopCompartmentAccepted EventKind = "cashshop.compartment_accepted"
	EventKindCashShopCompartmentReleased EventKind = "cashshop.compartment_released"
	EventKindCashShopCompartmentError    EventKind = "cashshop.compartment_error"

	// MTS custody (atlas-mts custody acks on EVENT_TOPIC_MTS_CUSTODY_STATUS).
	EventKindMtsCustodyAccepted EventKind = "mts.custody_accepted"
	EventKindMtsCustodyReleased EventKind = "mts.custody_released"
	EventKindMtsCustodyMoved    EventKind = "mts.custody_moved"
	EventKindMtsCustodyError    EventKind = "mts.custody_error"

	// Compartment (character inventory).
	EventKindCompartmentCreated        EventKind = "compartment.created"
	EventKindCompartmentCreationFailed EventKind = "compartment.creation_failed"
	EventKindCompartmentDeleted        EventKind = "compartment.deleted"
	EventKindCompartmentAccepted       EventKind = "compartment.accepted"
	EventKindCompartmentReleased       EventKind = "compartment.released"
	EventKindCompartmentError          EventKind = "compartment.error"

	// Inventory (rollup of all compartments for a character).
	EventKindInventoryCreated        EventKind = "inventory.created"
	EventKindInventoryCreationFailed EventKind = "inventory.creation_failed"

	// Storage.
	EventKindStorageMesosUpdated        EventKind = "storage.mesos_updated"
	EventKindStorageError               EventKind = "storage.error"
	EventKindStorageCompartmentAccepted EventKind = "storage.compartment_accepted"
	EventKindStorageCompartmentReleased EventKind = "storage.compartment_released"
	EventKindStorageCompartmentError    EventKind = "storage.compartment_error"

	// Guild.
	EventKindGuildRequestAgreement EventKind = "guild.request_agreement"
	EventKindGuildCreated          EventKind = "guild.created"
	EventKindGuildDisbanded        EventKind = "guild.disbanded"
	EventKindGuildEmblemUpdated    EventKind = "guild.emblem_updated"
	EventKindGuildCapacityUpdated  EventKind = "guild.capacity_updated"

	// Invite.
	EventKindInviteCreated  EventKind = "invite.created"
	EventKindInviteAccepted EventKind = "invite.accepted"
	EventKindInviteRejected EventKind = "invite.rejected"
)

// acceptanceTable maps each saga.Action to the set of EventKinds that can
// complete (or fail) a step of that action. Empty slice means self-completing
// — no Kafka event advances the step. A missing entry is a bug: unknown
// actions default-deny in StepAcceptsEvent, but the coverage test
// (event_acceptance_test.go) catches missing entries before runtime.
var acceptanceTable = map[sharedsaga.Action][]EventKind{
	// Asset actions.
	sharedsaga.AwardAsset:           {EventKindAssetCreated, EventKindAssetQuantityChanged},
	sharedsaga.DestroyAsset:         {EventKindAssetDeleted, EventKindAssetQuantityChanged},
	sharedsaga.DestroyAssetFromSlot: {EventKindAssetDeleted, EventKindAssetQuantityChanged},
	sharedsaga.EquipAsset:           {EventKindAssetMoved},
	sharedsaga.UnequipAsset:         {EventKindAssetMoved},
	sharedsaga.CreateAndEquipAsset:  {EventKindAssetCreated},

	// Character/stat actions.
	sharedsaga.AwardExperience:        {EventKindCharacterExperienceChanged},
	sharedsaga.AwardLevel:             {EventKindCharacterLevelChanged},
	sharedsaga.AwardMesos:             {EventKindCharacterMesoChanged, EventKindCharacterMesoError},
	sharedsaga.AwardCurrency:          {EventKindCashShopWalletUpdated, EventKindCashShopWalletError},
	sharedsaga.AwardFame:              {EventKindCharacterStatChanged},
	sharedsaga.ChangeJob:              {EventKindCharacterJobChanged},
	sharedsaga.ChangeHair:             {EventKindCharacterStatChanged},
	sharedsaga.ChangeFace:             {EventKindCharacterStatChanged},
	sharedsaga.ChangeSkin:             {EventKindCharacterStatChanged},
	sharedsaga.SetHP:                  {EventKindCharacterStatChanged},
	sharedsaga.DeductExperience:       {EventKindCharacterExperienceChanged},
	sharedsaga.CancelAllBuffs:         {EventKindCharacterStatChanged},
	sharedsaga.ResetStats:             {EventKindCharacterStatChanged},
	sharedsaga.RebalanceAP:            {EventKindCharacterStatChanged},
	sharedsaga.TransferAP:             {EventKindCharacterStatChanged, EventKindCharacterApTransferError},
	sharedsaga.TransferSP:             {EventKindSkillSpTransferred, EventKindSkillSpTransferError},
	sharedsaga.ValidateCharacterState: {},
	sharedsaga.IncreaseBuddyCapacity:  {EventKindBuddyCapacityChanged},
	sharedsaga.GainCloseness:          {EventKindPetClosenessChanged},
	sharedsaga.EvolvePet:              {EventKindPetEvolved},

	// Skills.
	sharedsaga.CreateSkill: {EventKindSkillCreated},
	sharedsaga.UpdateSkill: {EventKindSkillUpdated},

	// Quest.
	sharedsaga.CompleteQuest:    {EventKindQuestCompleted},
	sharedsaga.StartQuest:       {EventKindQuestStarted},
	sharedsaga.SetQuestProgress: {EventKindQuestStarted},
	sharedsaga.ForfeitQuest:     {EventKindQuestForfeited},

	// Consumable.
	sharedsaga.ApplyConsumableEffect:  {EventKindConsumableEffectApplied},
	sharedsaga.CancelConsumableEffect: {},

	// Storage.
	sharedsaga.ShowStorage:          {},
	sharedsaga.DepositToStorage:     {EventKindCompartmentAccepted, EventKindCompartmentError},
	sharedsaga.UpdateStorageMesos:   {EventKindStorageMesosUpdated, EventKindStorageError},
	sharedsaga.TransferToStorage:    {}, // composite: expanded into sub-steps before dispatch
	sharedsaga.WithdrawFromStorage:  {}, // composite
	sharedsaga.AcceptToStorage:      {EventKindStorageCompartmentAccepted, EventKindStorageCompartmentError},
	sharedsaga.ReleaseFromCharacter: {EventKindCompartmentReleased, EventKindCompartmentError},
	sharedsaga.AcceptToCharacter:    {EventKindCompartmentAccepted, EventKindCompartmentError},
	sharedsaga.ReleaseFromStorage:   {EventKindStorageCompartmentReleased, EventKindStorageCompartmentError},

	// Cash shop.
	sharedsaga.TransferToCashShop:   {}, // composite
	sharedsaga.WithdrawFromCashShop: {}, // composite
	sharedsaga.AcceptToCashShop:     {EventKindCashShopCompartmentAccepted, EventKindCashShopCompartmentError},
	sharedsaga.ReleaseFromCashShop:  {EventKindCashShopCompartmentReleased, EventKindCashShopCompartmentError},

	// MTS.
	sharedsaga.TransferToMts:           {}, // composite: expanded into release_from_character + accept_to_mts_listing
	sharedsaga.WithdrawFromMts:         {}, // composite: expanded into release_from_mts_holding + accept_to_character
	sharedsaga.MtsSettlePurchase:       {}, // composite: expanded into award_currency×2 + mts_move_listing_to_holding
	sharedsaga.AcceptToMtsListing:      {EventKindMtsCustodyAccepted, EventKindMtsCustodyError},
	sharedsaga.ReleaseFromMtsHolding:   {EventKindMtsCustodyReleased, EventKindMtsCustodyError},
	sharedsaga.MtsMoveListingToHolding: {EventKindMtsCustodyMoved, EventKindMtsCustodyError},
	sharedsaga.MtsBidEscrow:            {EventKindCashShopWalletUpdated, EventKindCashShopWalletError}, // reuses the cash-shop wallet ack

	// Guild.
	sharedsaga.RequestGuildName:             {EventKindGuildRequestAgreement, EventKindGuildCreated},
	sharedsaga.RequestGuildEmblem:           {EventKindGuildEmblemUpdated},
	sharedsaga.RequestGuildDisband:          {EventKindGuildDisbanded},
	sharedsaga.RequestGuildCapacityIncrease: {EventKindGuildCapacityUpdated},

	// Invite.
	sharedsaga.CreateInvite: {EventKindInviteCreated, EventKindInviteAccepted, EventKindInviteRejected},

	// Character lifecycle.
	sharedsaga.CreateCharacter:       {EventKindCharacterCreated, EventKindCharacterCreationFailed},
	sharedsaga.AwaitCharacterCreated: {EventKindCharacterCreated, EventKindCharacterCreationFailed},
	sharedsaga.AwaitInventoryCreated: {EventKindInventoryCreated, EventKindInventoryCreationFailed},

	// Fire-and-forget / self-completing actions (no Kafka event advances them).
	sharedsaga.WarpToRandomPortal:         {},
	sharedsaga.WarpToPortal:               {},
	sharedsaga.WarpToSavedLocation:        {},
	sharedsaga.SaveLocation:               {},
	sharedsaga.SendMessage:                {},
	sharedsaga.FieldEffect:                {},
	sharedsaga.FieldEffectWeather:         {},
	sharedsaga.UiLock:                     {},
	sharedsaga.PlayPortalSound:            {},
	sharedsaga.UpdateAreaInfo:             {},
	sharedsaga.ShowInfo:                   {},
	sharedsaga.ShowInfoText:               {},
	sharedsaga.ShowIntro:                  {},
	sharedsaga.ShowHint:                   {},
	sharedsaga.ShowGuideHint:              {},
	sharedsaga.BlockPortal:                {},
	sharedsaga.UnblockPortal:              {},
	sharedsaga.SpawnMonster:               {},
	sharedsaga.SpawnReactorDrops:          {},
	sharedsaga.HitReactor:                 {},
	sharedsaga.BroadcastPqMessage:         {},
	sharedsaga.RegisterPartyQuest:         {},
	sharedsaga.WarpPartyQuestMembersToMap: {},
	sharedsaga.LeavePartyQuest:            {},
	sharedsaga.EnterPartyQuestBonus:       {},
	sharedsaga.UpdatePqCustomData:         {},
	sharedsaga.StageClearAttemptPq:        {},
	sharedsaga.SelectGachaponReward:       {},
	sharedsaga.EmitGachaponWin:            {},
	sharedsaga.StartInstanceTransport:     {},
	sharedsaga.StartRPSGame:               {},
}

// StepAcceptsEvent reports whether a saga step's Action can be legitimately
// completed by an event of the given EventKind. Unknown actions default-deny.
func StepAcceptsEvent(action sharedsaga.Action, kind EventKind) bool {
	kinds, ok := acceptanceTable[action]
	if !ok {
		return false
	}
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// EventOutcome classifies an EventKind as a success signal (the step's side
// effect landed downstream) or a failure signal (it did not). Late-after-
// terminal routing (design §3.2/§3.4) uses this to decide whether a rollback
// must be dispatched for an absorbed event.
type EventOutcome string

const (
	OutcomeSuccess EventOutcome = "success"
	OutcomeFailure EventOutcome = "failure"
)

// outcomeTable classifies every declared EventKind. invite.rejected is a
// failure deliberately: a rejected invite left no side effect to roll back.
var outcomeTable = map[EventKind]EventOutcome{
	// Character subsystem.
	EventKindCharacterMapChanged:        OutcomeSuccess,
	EventKindCharacterExperienceChanged: OutcomeSuccess,
	EventKindCharacterLevelChanged:      OutcomeSuccess,
	EventKindCharacterMesoChanged:       OutcomeSuccess,
	EventKindCharacterJobChanged:        OutcomeSuccess,
	EventKindCharacterCreated:           OutcomeSuccess,
	EventKindCharacterCreationFailed:    OutcomeFailure,
	EventKindCharacterStatChanged:       OutcomeSuccess,
	EventKindCharacterApTransferError:   OutcomeFailure,
	EventKindCharacterMesoError:         OutcomeFailure,
	EventKindCharacterDeleted:           OutcomeSuccess,

	// Asset subsystem.
	EventKindAssetCreated:         OutcomeSuccess,
	EventKindAssetDeleted:         OutcomeSuccess,
	EventKindAssetQuantityChanged: OutcomeSuccess,
	EventKindAssetMoved:           OutcomeSuccess,

	// Quest subsystem.
	EventKindQuestStarted:   OutcomeSuccess,
	EventKindQuestCompleted: OutcomeSuccess,
	EventKindQuestForfeited: OutcomeSuccess,

	// Skill subsystem.
	EventKindSkillCreated:         OutcomeSuccess,
	EventKindSkillUpdated:         OutcomeSuccess,
	EventKindSkillDeleted:         OutcomeSuccess,
	EventKindSkillSpTransferred:   OutcomeSuccess,
	EventKindSkillSpTransferError: OutcomeFailure,

	// Buddy list.
	EventKindBuddyCapacityChanged: OutcomeSuccess,

	// Consumable.
	EventKindConsumableEffectApplied: OutcomeSuccess,

	// Pet.
	EventKindPetClosenessChanged: OutcomeSuccess,
	EventKindPetEvolved:          OutcomeSuccess,

	// Cash shop.
	EventKindCashShopWalletUpdated:       OutcomeSuccess,
	EventKindCashShopWalletError:         OutcomeFailure,
	EventKindCashShopCompartmentAccepted: OutcomeSuccess,
	EventKindCashShopCompartmentReleased: OutcomeSuccess,
	EventKindCashShopCompartmentError:    OutcomeFailure,

	// MTS custody (atlas-mts custody acks). A late success after a timeout must
	// be classified so the terminal-race late-compensation path can roll back
	// the custody move/accept/release; the *_error ack is a failure (no effect
	// landed, absorb only).
	EventKindMtsCustodyAccepted: OutcomeSuccess,
	EventKindMtsCustodyReleased: OutcomeSuccess,
	EventKindMtsCustodyMoved:    OutcomeSuccess,
	EventKindMtsCustodyError:    OutcomeFailure,

	// Compartment (character inventory).
	EventKindCompartmentCreated:        OutcomeSuccess,
	EventKindCompartmentCreationFailed: OutcomeFailure,
	EventKindCompartmentDeleted:        OutcomeSuccess,
	EventKindCompartmentAccepted:       OutcomeSuccess,
	EventKindCompartmentReleased:       OutcomeSuccess,
	EventKindCompartmentError:          OutcomeFailure,

	// Inventory.
	EventKindInventoryCreated:        OutcomeSuccess,
	EventKindInventoryCreationFailed: OutcomeFailure,

	// Storage.
	EventKindStorageMesosUpdated:        OutcomeSuccess,
	EventKindStorageError:               OutcomeFailure,
	EventKindStorageCompartmentAccepted: OutcomeSuccess,
	EventKindStorageCompartmentReleased: OutcomeSuccess,
	EventKindStorageCompartmentError:    OutcomeFailure,

	// Guild.
	EventKindGuildRequestAgreement: OutcomeSuccess,
	EventKindGuildCreated:          OutcomeSuccess,
	EventKindGuildDisbanded:        OutcomeSuccess,
	EventKindGuildEmblemUpdated:    OutcomeSuccess,
	EventKindGuildCapacityUpdated:  OutcomeSuccess,

	// Invite.
	EventKindInviteCreated:  OutcomeSuccess,
	EventKindInviteAccepted: OutcomeSuccess,
	EventKindInviteRejected: OutcomeFailure,
}

// EventOutcomeOf returns the outcome classification for kind.
func EventOutcomeOf(kind EventKind) (EventOutcome, bool) {
	o, ok := outcomeTable[kind]
	return o, ok
}

// SkipReason* constants are the `reason` field values on structured debug
// logs emitted when AcceptEvent (or a handler-level guard) refuses to
// complete a step. Centralised so per-consumer drift is impossible.
const (
	SkipReasonSagaNotFound       = "saga_not_found"
	SkipReasonNoPendingStep      = "no_pending_step"
	SkipReasonActionMismatch     = "action_mismatch"
	SkipReasonTemplateIdMismatch = "template_id_mismatch"
	SkipReasonUnmatchedEvent     = "unmatched_event"
	SkipReasonNilTransactionId   = "nil_transaction_id"
	SkipReasonSagaTerminal       = "saga_terminal"
)

// LogSkip emits a debug-level structured log with a `reason` field.
// Consumer packages may call this directly for handler-local skips
// (e.g., template-id mismatch in the asset handler).
func LogSkip(l logrus.FieldLogger, fields logrus.Fields, reason string) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["reason"] = reason
	l.WithFields(fields).Debug("Saga event skipped.")
}

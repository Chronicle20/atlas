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
	EventKindSkillCreated EventKind = "skill.created"
	EventKindSkillUpdated EventKind = "skill.updated"
	EventKindSkillDeleted EventKind = "skill.deleted"

	// Buddy list.
	EventKindBuddyCapacityChanged EventKind = "buddy.capacity_changed"

	// Consumable.
	EventKindConsumableEffectApplied EventKind = "consumable.effect_applied"

	// Pet.
	EventKindPetClosenessChanged EventKind = "pet.closeness_changed"

	// Cash shop.
	EventKindCashShopWalletUpdated       EventKind = "cashshop.wallet_updated"
	EventKindCashShopCompartmentAccepted EventKind = "cashshop.compartment_accepted"
	EventKindCashShopCompartmentReleased EventKind = "cashshop.compartment_released"
	EventKindCashShopCompartmentError    EventKind = "cashshop.compartment_error"

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
	sharedsaga.AwardCurrency:          {EventKindCashShopWalletUpdated},
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
	sharedsaga.ValidateCharacterState: {},
	sharedsaga.IncreaseBuddyCapacity:  {EventKindBuddyCapacityChanged},
	sharedsaga.GainCloseness:          {EventKindPetClosenessChanged},

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

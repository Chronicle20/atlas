package saga

import (
	"time"

	"github.com/google/uuid"
)

// Type the type of saga
type Type string

// Constants for different saga types
const (
	InventoryTransaction Type = "inventory_transaction"
	QuestReward          Type = "quest_reward"
	TradeTransaction     Type = "trade_transaction"
	// TradeStaging is one item moving into trade escrow (transfer_to_trade).
	// It is deliberately NOT TradeTransaction: a settlement is a two-party swap
	// whose reverse-walk pairs releases with accepts by asset id, while a stage
	// is the single-item release+accept shape MtsOperation already models. Reusing
	// TradeTransaction would send a staging failure through the swap's
	// pairing logic, find no paired accept, and silently skip the re-grant —
	// destroying the item (task-205 design §5A.4).
	TradeStaging          Type = "trade_staging"
	CharacterCreation     Type = "character_creation"
	StorageOperation      Type = "storage_operation"
	CashShopOperation     Type = "cash_shop_operation"
	CharacterRespawn      Type = "character_respawn"
	GachaponTransaction   Type = "gachapon_transaction"
	MtsOperation          Type = "mts_operation"
	FieldEffectUse        Type = "field_effect_use"
	TeleportRockUse       Type = "teleport_rock_use"
	QuestStart            Type = "quest_start"
	QuestComplete         Type = "quest_complete"
	QuestRestoreItem      Type = "quest_restore_item"
	PetEvolution          Type = "pet_evolution"
	NoteSend              Type = "note_send"
	SkillBookUse          Type = "skill_book_use"
	ItemTagUse            Type = "item_tag_use"
	SealingLockUse        Type = "sealing_lock_use"
	KarmaScissorsUse      Type = "karma_scissors_use"
	IncubatorUse          Type = "incubator_use"
	ExpirationExtenderUse Type = "expiration_extender_use"
	PointReset            Type = "point_reset"
	MegaphoneUse          Type = "megaphone_use"
	MesoSackUse           Type = "meso_sack_use"
	PetNameTagUse         Type = "pet_name_tag_use"
	// RemoteMerchant is the classification-545 cash item flow: open an NPC's
	// shop from anywhere, then consume the item — never the other way round
	// (task-221).
	RemoteMerchant Type = "remote_merchant"
	// WorldTransfer moves a character between worlds. Its steps run in a fixed,
	// load-bearing order: validate -> leave_guild -> leave_party -> sever_buddies
	// -> change_character_world. The world update is last and is a single-row
	// update, so a failure anywhere leaves the character wholly in the source
	// world with only recoverable severances applied — the character is never
	// in two worlds and never in none.
	WorldTransfer Type = "world_transfer"

	// PetRevive is the classification-518 Water of Life flow: consume the item,
	// then reset a dried-up pet's lifespan. Consume comes first so a failed
	// revive compensates into a refund rather than a free revive (task-228).
	PetRevive Type = "pet_revive"

	// ScriptedItemUse is the classification-243 flow: open the item's own
	// dialogue, then consume the item — in that order, so an unauthored item
	// costs the player nothing.
	ScriptedItemUse Type = "scripted_item_use"

	// RemoteNpcUse is the classification-239 flow: open the named NPC's shop or
	// conversation from anywhere, then consume the item.
	RemoteNpcUse Type = "remote_npc_use"

	// ParcelSend and ParcelReceive are the Duey parcel-delivery sagas
	// (task-241): ParcelSend moves an item/meso bundle into custody at send
	// time, ParcelReceive moves it out of custody to the recipient (or back to
	// the sender on discard/expiry).
	ParcelSend    Type = "parcel_send"
	ParcelReceive Type = "parcel_receive"

	// MapleLifeUse is the classification-543 flow: create the character through
	// atlas-character-factory FIRST, then destroy the cash item once the seed
	// saga reports CREATED. One step, so there is nothing to reverse-walk — the
	// item survives every failure by construction (task-246 design §5.4).
	MapleLifeUse Type = "maple_life_use"
)

// Status represents the status of a saga step
type Status string

const (
	Pending   Status = "pending"
	Completed Status = "completed"
	Failed    Status = "failed"
)

// Action represents an action type for saga steps
type Action string

// Constants for different actions
const (
	// Core inventory/asset actions
	AwardAsset           Action = "award_asset"
	AwardExperience      Action = "award_experience"
	AwardLevel           Action = "award_level"
	AwardMesos           Action = "award_mesos"
	AwardCurrency        Action = "award_currency"
	AwardFame            Action = "award_fame"
	DestroyAsset         Action = "destroy_asset"
	DestroyAllAssets     Action = "destroy_all_assets"
	DestroyAssetFromSlot Action = "destroy_asset_from_slot"
	EquipAsset           Action = "equip_asset"
	UnequipAsset         Action = "unequip_asset"
	CreateAndEquipAsset  Action = "create_and_equip_asset"

	// Warp actions
	WarpToRandomPortal  Action = "warp_to_random_portal"
	WarpToPortal        Action = "warp_to_portal"
	WarpToSavedLocation Action = "warp_to_saved_location"
	SaveLocation        Action = "save_location"

	// Character state actions
	ChangeJob              Action = "change_job"
	ChangeHair             Action = "change_hair"
	ChangeFace             Action = "change_face"
	ChangeSkin             Action = "change_skin"
	SetHP                  Action = "set_hp"
	DeductExperience       Action = "deduct_experience"
	CancelAllBuffs         Action = "cancel_all_buffs"
	ResetStats             Action = "reset_stats"
	RebalanceAP            Action = "rebalance_ap"
	ValidateCharacterState Action = "validate_character_state"
	IncreaseBuddyCapacity  Action = "increase_buddy_capacity"
	GainCloseness          Action = "gain_closeness"
	EvolvePet              Action = "evolve_pet"
	RevivePet              Action = "revive_pet"
	RenamePet              Action = "rename_pet"
	TransferAP             Action = "transfer_ap"
	TransferSP             Action = "transfer_sp"

	// Skill actions
	CreateSkill Action = "create_skill"
	UpdateSkill Action = "update_skill"

	// Quest actions
	CompleteQuest    Action = "complete_quest"
	StartQuest       Action = "start_quest"
	SetQuestProgress Action = "set_quest_progress"
	ForfeitQuest     Action = "forfeit_quest"

	// Consumable effect actions
	ApplyConsumableEffect  Action = "apply_consumable_effect"
	CancelConsumableEffect Action = "cancel_consumable_effect"

	// Message actions
	SendMessage Action = "send_message"

	// UI/visual effect actions
	FieldEffect     Action = "field_effect"
	UiLock          Action = "ui_lock"
	PlayPortalSound Action = "play_portal_sound"
	UpdateAreaInfo  Action = "update_area_info"
	ShowInfo        Action = "show_info"
	ShowInfoText    Action = "show_info_text"
	ShowIntro       Action = "show_intro"
	ShowHint        Action = "show_hint"
	ShowGuideHint   Action = "show_guide_hint"
	BlockPortal     Action = "block_portal"
	UnblockPortal   Action = "unblock_portal"

	// Spawn actions
	SpawnMonster      Action = "spawn_monster"
	SpawnReactorDrops Action = "spawn_reactor_drops"

	// Storage actions
	ShowStorage          Action = "show_storage"
	DepositToStorage     Action = "deposit_to_storage"
	UpdateStorageMesos   Action = "update_storage_mesos"
	TransferToStorage    Action = "transfer_to_storage"
	WithdrawFromStorage  Action = "withdraw_from_storage"
	AcceptToStorage      Action = "accept_to_storage"
	ReleaseFromCharacter Action = "release_from_character"
	AcceptToCharacter    Action = "accept_to_character"
	ReleaseFromStorage   Action = "release_from_storage"

	// NPC shop actions
	OpenNpcShop Action = "open_npc_shop"

	// NPC conversation actions. Both are deliberately NOT self-completing: the
	// step stays Pending until EVENT_TOPIC_NPC_CONVERSATION_STATUS reports
	// STARTED or START_ERROR, which is what lets a following destroy step
	// consume the item only once the dialogue actually opened.
	//
	// Two discrete actions rather than one with a mode discriminator: the
	// orchestrator's handler dispatch is per-action, and a discriminator inside
	// the payload would move branching somewhere the compensator and
	// event-acceptance tables cannot see it.
	StartItemConversation Action = "start_item_conversation"
	StartNpcConversation  Action = "start_npc_conversation"

	// Trade actions (task-205). trade_settlement is a COMPOSITE: the
	// orchestrator expands it into release_from_character / accept_to_character /
	// award_mesos steps (see expandTradeSettlement). atlas-trades never
	// enumerates concrete saga steps itself. The saga type is the pre-existing
	// TradeTransaction.
	TradeSettlement Action = "trade_settlement"

	// TradeUnwind is the teardown twin of TradeSettlement: a COMPOSITE that
	// returns an abandoned trade's escrow to the people it came from.
	TradeUnwind Action = "trade_unwind"

	// Trade escrow custody (task-205, design §5A.2). transfer_to_trade is a
	// COMPOSITE expanded into release_from_character + accept_to_trade, the
	// same shape as transfer_to_mts; accept_to_trade and release_from_trade are
	// the atomic custody steps dispatched to atlas-trades. A staged item leaves
	// the owner's compartment at STAGE time — the inventory delta that results
	// is what clears the client's m_bExclRequestSent, which nothing else in the
	// trade flow does (design §5A.1).
	TransferToTrade  Action = "transfer_to_trade"
	AcceptToTrade    Action = "accept_to_trade"
	ReleaseFromTrade Action = "release_from_trade"

	// Cash shop actions
	TransferToCashShop   Action = "transfer_to_cash_shop"
	WithdrawFromCashShop Action = "withdraw_from_cash_shop"
	AcceptToCashShop     Action = "accept_to_cash_shop"
	ReleaseFromCashShop  Action = "release_from_cash_shop"

	// MTS marketplace
	TransferToMts           Action = "transfer_to_mts"
	WithdrawFromMts         Action = "withdraw_from_mts"
	AcceptToMtsListing      Action = "accept_to_mts_listing"
	ReleaseFromMtsHolding   Action = "release_from_mts_holding"
	MtsSettlePurchase       Action = "mts_settle_purchase"
	MtsMoveListingToHolding Action = "mts_move_listing_to_holding"
	MtsBidEscrow            Action = "mts_bid_escrow"

	// Parcel custody (task-241). transfer_to_parcel is a COMPOSITE expanded into
	// release_from_character + accept_to_parcel, the same shape as transfer_to_mts.
	TransferToParcel   Action = "transfer_to_parcel"
	AcceptToParcel     Action = "accept_to_parcel"
	ReleaseFromParcel  Action = "release_from_parcel"
	WithdrawFromParcel Action = "withdraw_from_parcel"

	// ShowParcel opens the Duey dialog for a character, like ShowStorage opens
	// the storage UI. Self-completing: nothing downstream depends on the dialog
	// having opened, because the ticket is consumed by the parcel_send saga and
	// not by opening the interface (task-241 FR-26).
	ShowParcel Action = "show_parcel"

	// Guild actions
	RequestGuildName             Action = "request_guild_name"
	RequestGuildEmblem           Action = "request_guild_emblem"
	RequestGuildDisband          Action = "request_guild_disband"
	RequestGuildCapacityIncrease Action = "request_guild_capacity_increase"
	CreateInvite                 Action = "create_invite"

	// Character creation actions
	CreateCharacter       Action = "create_character"
	AwaitCharacterCreated Action = "await_character_created"
	AwaitInventoryCreated Action = "await_inventory_created"

	// Transport actions
	StartInstanceTransport Action = "start_instance_transport"

	// Gachapon actions
	SelectGachaponReward Action = "select_gachapon_reward"
	EmitGachaponWin      Action = "emit_gachapon_win"

	// RPS actions
	StartRPSGame Action = "start_rps_game"

	// Party quest actions
	RegisterPartyQuest         Action = "register_party_quest"
	WarpPartyQuestMembersToMap Action = "warp_party_quest_members_to_map"
	LeavePartyQuest            Action = "leave_party_quest"
	EnterPartyQuestBonus       Action = "enter_party_quest_bonus"

	// Party quest reactor orchestration actions
	UpdatePqCustomData  Action = "update_pq_custom_data"
	HitReactor          Action = "hit_reactor"
	BroadcastPqMessage  Action = "broadcast_pq_message"
	StageClearAttemptPq Action = "stage_clear_attempt_pq"

	// Field effect actions
	FieldEffectWeather Action = "field_effect_weather"
	// PlayJukebox starts a song in one field. DurationMs is the client's own
	// IWzSound::length; atlas-maps caps it. The BGM name is never carried --
	// the client resolves it from the item's WZ info/path node itself.
	PlayJukebox Action = "play_jukebox"

	// Environment object actions. Both are fire-and-forget: the step
	// completes when the command is produced, and neither has a
	// compensating action -- reversing a move is the script author's job
	// (a second move_environment, or reset_environment).
	MoveEnvironment  Action = "move_environment"
	ResetEnvironment Action = "reset_environment"

	// Note actions
	CreateNote Action = "create_note"

	// Item tag / sealing lock / incubator actions
	SetAssetOwner   Action = "set_asset_owner"
	ApplyAssetLock  Action = "apply_asset_lock"
	ApplyAssetKarma Action = "apply_asset_karma"
	// ExtendAssetExpiration pushes a time-limited asset's expiration out. It
	// is deliberately NOT ApplyAssetLock: that action stamps FlagLock and
	// rejects an unlocked asset carrying a non-zero expiration, which is
	// exactly this action's only valid target.
	ExtendAssetExpiration Action = "extend_asset_expiration"
	IncubatorResult       Action = "incubator_result"

	// Megaphone / world broadcast actions
	EmitMegaphone         Action = "emit_megaphone"
	EnqueueWorldBroadcast Action = "enqueue_world_broadcast"

	// World transfer actions (task-227). Order is fixed and load-bearing:
	// validate -> leave_guild -> leave_party -> sever_buddies ->
	// change_character_world. See the WorldTransfer Type doc comment.
	ValidateWorldTransfer   Action = "validate_world_transfer"
	LeaveGuildForTransfer   Action = "leave_guild_for_transfer"
	LeavePartyForTransfer   Action = "leave_party_for_transfer"
	SeverBuddiesForTransfer Action = "sever_buddies_for_transfer"
	ChangeCharacterWorld    Action = "change_character_world"
)

// Saga represents the entire saga transaction.
type Saga struct {
	TransactionId uuid.UUID   `json:"transactionId"`     // Unique ID for the transaction
	SagaType      Type        `json:"sagaType"`          // Type of the saga (e.g., inventory_transaction)
	InitiatedBy   string      `json:"initiatedBy"`       // Who initiated the saga (e.g., NPC ID, user)
	Timeout       int64       `json:"timeout,omitempty"` // Optional per-saga timeout in milliseconds; 0 → orchestrator default (30s)
	Steps         []Step[any] `json:"steps"`             // List of steps in the saga
}

// Failing returns true if any step has failed status
func (s *Saga) Failing() bool {
	for _, step := range s.Steps {
		if step.Status == Failed {
			return true
		}
	}
	return false
}

// GetCurrentStep returns the first pending step
func (s *Saga) GetCurrentStep() (Step[any], bool) {
	for idx, step := range s.Steps {
		if step.Status == Pending {
			return s.Steps[idx], true
		}
	}
	return Step[any]{}, false
}

// FindFurthestCompletedStepIndex returns the index of the furthest completed step (last one with status "completed")
// Returns -1 if no completed step is found
func (s *Saga) FindFurthestCompletedStepIndex() int {
	furthestCompletedIndex := -1
	for i := len(s.Steps) - 1; i >= 0; i-- {
		if s.Steps[i].Status == Completed {
			furthestCompletedIndex = i
			break
		}
	}
	return furthestCompletedIndex
}

// FindEarliestPendingStepIndex returns the index of the earliest pending step (first one with status "pending")
// Returns -1 if no pending step is found
func (s *Saga) FindEarliestPendingStepIndex() int {
	for i := 0; i < len(s.Steps); i++ {
		if s.Steps[i].Status == Pending {
			return i
		}
	}
	return -1
}

// FindFailedStepIndex returns the index of the first failed step
// Returns -1 if no failed step is found
func (s *Saga) FindFailedStepIndex() int {
	for i := 0; i < len(s.Steps); i++ {
		if s.Steps[i].Status == Failed {
			return i
		}
	}
	return -1
}

// SetStepStatus sets the status of a step at the given index
func (s *Saga) SetStepStatus(index int, status Status) {
	if index >= 0 && index < len(s.Steps) {
		s.Steps[index].Status = status
	}
}

// Step represents a single step within a saga.
type Step[T any] struct {
	StepId    string    `json:"stepId"`    // Unique ID for the step
	Status    Status    `json:"status"`    // Status of the step (e.g., pending, completed, failed)
	Action    Action    `json:"action"`    // The Action to be taken
	Payload   T         `json:"payload"`   // Data required for the action (specific to the action type)
	CreatedAt time.Time `json:"createdAt"` // Timestamp of when the step was created
	UpdatedAt time.Time `json:"updatedAt"` // Timestamp of the last update to the step
}

// ExperienceDistributions represents how experience is distributed
type ExperienceDistributions struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

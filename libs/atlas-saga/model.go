package saga

import (
	"time"

	"github.com/google/uuid"
)

// Type the type of saga
type Type string

// Constants for different saga types
const (
	InventoryTransaction      Type = "inventory_transaction"
	QuestReward               Type = "quest_reward"
	TradeTransaction          Type = "trade_transaction"
	CharacterCreation Type = "character_creation"
	StorageOperation  Type = "storage_operation"
	CashShopOperation         Type = "cash_shop_operation"
	CharacterRespawn          Type = "character_respawn"
	GachaponTransaction       Type = "gachapon_transaction"
	FieldEffectUse            Type = "field_effect_use"
	QuestStart                Type = "quest_start"
	QuestComplete             Type = "quest_complete"
	QuestRestoreItem          Type = "quest_restore_item"
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
	ValidateCharacterState Action = "validate_character_state"
	IncreaseBuddyCapacity  Action = "increase_buddy_capacity"
	GainCloseness          Action = "gain_closeness"

	// Skill actions
	CreateSkill Action = "create_skill"
	UpdateSkill Action = "update_skill"

	// Quest actions
	CompleteQuest    Action = "complete_quest"
	StartQuest       Action = "start_quest"
	SetQuestProgress Action = "set_quest_progress"

	// Consumable effect actions
	ApplyConsumableEffect  Action = "apply_consumable_effect"
	CancelConsumableEffect Action = "cancel_consumable_effect"

	// Message actions
	SendMessage Action = "send_message"

	// UI/visual effect actions
	FieldEffect    Action = "field_effect"
	UiLock         Action = "ui_lock"
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
	ShowStorage        Action = "show_storage"
	DepositToStorage   Action = "deposit_to_storage"
	UpdateStorageMesos Action = "update_storage_mesos"
	TransferToStorage  Action = "transfer_to_storage"
	WithdrawFromStorage Action = "withdraw_from_storage"
	AcceptToStorage     Action = "accept_to_storage"
	ReleaseFromCharacter Action = "release_from_character"
	AcceptToCharacter    Action = "accept_to_character"
	ReleaseFromStorage   Action = "release_from_storage"

	// Cash shop actions
	TransferToCashShop   Action = "transfer_to_cash_shop"
	WithdrawFromCashShop Action = "withdraw_from_cash_shop"
	AcceptToCashShop     Action = "accept_to_cash_shop"
	ReleaseFromCashShop  Action = "release_from_cash_shop"

	// Guild actions
	RequestGuildName             Action = "request_guild_name"
	RequestGuildEmblem           Action = "request_guild_emblem"
	RequestGuildDisband          Action = "request_guild_disband"
	RequestGuildCapacityIncrease Action = "request_guild_capacity_increase"
	CreateInvite                 Action = "create_invite"

	// Character creation actions
	CreateCharacter       Action = "create_character"
	AwaitCharacterCreated Action = "await_character_created"

	// Transport actions
	StartInstanceTransport Action = "start_instance_transport"

	// Gachapon actions
	SelectGachaponReward Action = "select_gachapon_reward"
	EmitGachaponWin      Action = "emit_gachapon_win"

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
)

// Saga represents the entire saga transaction.
type Saga struct {
	TransactionId uuid.UUID   `json:"transactionId"` // Unique ID for the transaction
	SagaType      Type        `json:"sagaType"`      // Type of the saga (e.g., inventory_transaction)
	InitiatedBy   string      `json:"initiatedBy"`   // Who initiated the saga (e.g., NPC ID, user)
	Steps         []Step[any] `json:"steps"`         // List of steps in the saga
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
